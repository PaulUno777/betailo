import { PaymentMethod } from '../enums';

export const PAYMENT_METHOD_LABEL_KEYS: Readonly<Record<PaymentMethod, string>> = {
  mtn_momo: 'paymentMethod.mtnMomo',
  orange_money: 'paymentMethod.orangeMoney',
  card: 'paymentMethod.card'
};

export const PAYMENT_METHOD_VALUES: readonly PaymentMethod[] = Object.keys(PAYMENT_METHOD_LABEL_KEYS) as PaymentMethod[];
