/** The browser's passkey API, fed with the JSON the server exchanges.
 * The server speaks base64url for every binary field; `navigator.
 * credentials` wants ArrayBuffers and gives ArrayBuffers back. */

function toBuffer(value: string): ArrayBuffer {
  const base64 = value.replace(/-/g, '+').replace(/_/g, '/');
  const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4);
  const raw = atob(padded);
  const bytes = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i++) bytes[i] = raw.charCodeAt(i);
  return bytes.buffer;
}

function toBase64Url(buffer: ArrayBuffer | null | undefined): string | undefined {
  if (!buffer) return undefined;
  let raw = '';
  for (const byte of new Uint8Array(buffer)) raw += String.fromCharCode(byte);
  return btoa(raw).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

interface Descriptor {
  id: string;
  type: string;
  transports?: string[];
}

function descriptors(list: unknown): PublicKeyCredentialDescriptor[] | undefined {
  if (!Array.isArray(list)) return undefined;
  return (list as Descriptor[]).map((d) => ({ ...d, id: toBuffer(d.id) }) as PublicKeyCredentialDescriptor);
}

export function passkeysSupported(): boolean {
  return typeof window !== 'undefined' && typeof window.PublicKeyCredential === 'function' && !!navigator.credentials;
}

/** Creates a passkey from the server's registration options and returns
 * the answer in the JSON form the server reads. */
export async function createPasskey(publicKey: Record<string, unknown>): Promise<unknown> {
  const user = publicKey['user'] as { id: string; name: string; displayName: string };
  const options = {
    ...publicKey,
    challenge: toBuffer(publicKey['challenge'] as string),
    user: { ...user, id: toBuffer(user.id) },
    excludeCredentials: descriptors(publicKey['excludeCredentials'])
  } as unknown as PublicKeyCredentialCreationOptions;
  const credential = (await navigator.credentials.create({ publicKey: options })) as PublicKeyCredential | null;
  if (!credential) throw new Error('no credential');
  const response = credential.response as AuthenticatorAttestationResponse;
  return {
    id: credential.id,
    rawId: toBase64Url(credential.rawId),
    type: credential.type,
    authenticatorAttachment: credential.authenticatorAttachment ?? undefined,
    clientExtensionResults: credential.getClientExtensionResults(),
    response: {
      clientDataJSON: toBase64Url(response.clientDataJSON),
      attestationObject: toBase64Url(response.attestationObject),
      transports: typeof response.getTransports === 'function' ? response.getTransports() : undefined
    }
  };
}

/** Signs a sign-in with a passkey from the server's request options. */
export async function usePasskey(publicKey: Record<string, unknown>): Promise<unknown> {
  const options = {
    ...publicKey,
    challenge: toBuffer(publicKey['challenge'] as string),
    allowCredentials: descriptors(publicKey['allowCredentials'])
  } as unknown as PublicKeyCredentialRequestOptions;
  const credential = (await navigator.credentials.get({ publicKey: options })) as PublicKeyCredential | null;
  if (!credential) throw new Error('no credential');
  const response = credential.response as AuthenticatorAssertionResponse;
  return {
    id: credential.id,
    rawId: toBase64Url(credential.rawId),
    type: credential.type,
    authenticatorAttachment: credential.authenticatorAttachment ?? undefined,
    clientExtensionResults: credential.getClientExtensionResults(),
    response: {
      clientDataJSON: toBase64Url(response.clientDataJSON),
      authenticatorData: toBase64Url(response.authenticatorData),
      signature: toBase64Url(response.signature),
      userHandle: toBase64Url(response.userHandle)
    }
  };
}
