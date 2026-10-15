import { Locale } from '../../core/i18n/locale.service';

/** One article of the terms. `id` is its anchor (`/terms#paiement`). */
export interface TermsSection {
  readonly id: string;
  readonly title: string;
  readonly paragraphs: readonly string[];
}

/** The general terms of use and sale. Long legal prose stays here rather
 * than in the translation files. Every rule stated is one the API
 * enforces: change the text when the rule changes (the fee in
 * reservation/reserve_cycle.go, the 48 h release in auto_release.go,
 * the 3 reports in 7 days in report/create_report.go). */
export const TERMS_UPDATED = '2026-10-15';

export const TERMS: Readonly<Record<Locale, readonly TermsSection[]>> = {
  fr: [
    {
      id: 'objet',
      title: 'Objet',
      paragraphs: [
        'Betailo est une place de marché qui met en relation des éleveurs et des acheteurs pour réserver des animaux issus de cycles d’élevage (poussins, porcelets, cabris…) ou disponibles à la vente.',
        'Le contrat de vente des animaux est conclu directement entre l’éleveur et l’acheteur. Betailo n’est ni l’éleveur ni le vendeur : la plateforme fournit l’outil de réservation, conserve le paiement jusqu’à la livraison et arbitre les litiges.',
        'En créant un compte, vous acceptez les présentes conditions générales. Si vous ne les acceptez pas, n’utilisez pas la plateforme.'
      ]
    },
    {
      id: 'compte',
      title: 'Votre compte',
      paragraphs: [
        'Vous vous engagez à fournir des informations exactes et à les tenir à jour. Un compte est personnel : vous êtes responsable de ce qui est fait avec, et vous gardez votre mot de passe secret.',
        'Betailo peut suspendre ou fermer un compte qui enfreint ces conditions, en particulier en cas de fraude ou de tentative d’arnaque.'
      ]
    },
    {
      id: 'eleveurs',
      title: 'Fermes et éleveurs',
      paragraphs: [
        'Tout titulaire d’un compte peut enregistrer une ou plusieurs fermes. Une ferme n’est visible sur la place de marché qu’après validation par l’équipe Betailo, sur la base des justificatifs fournis.',
        'L’éleveur garantit que ses annonces sont exactes : espèce, nombre, âge, prix, dates, état de santé et carnet de vaccination. Il détient les droits sur les photos publiées et respecte la réglementation sanitaire et vétérinaire applicable.',
        'L’éleveur s’engage à livrer des animaux conformes à l’annonce, dans les délais annoncés.'
      ]
    },
    {
      id: 'reservations',
      title: 'Réservations',
      paragraphs: [
        'Une réservation porte sur une quantité d’animaux d’une annonce. Le prix unitaire est figé au moment de la réservation, même si l’éleveur modifie son prix ensuite.',
        'Une réservation non payée place l’acheteur sur la liste d’attente du cycle : elle ne bloque aucune place, l’acheteur peut l’annuler à tout moment, et elle est annulée automatiquement si le cycle se termine avant le paiement.',
        'Seul le paiement bloque des places : un acheteur qui a payé est toujours servi en priorité.'
      ]
    },
    {
      id: 'paiement',
      title: 'Paiement et séquestre',
      paragraphs: [
        'Le paiement se fait sur la plateforme, par MTN Mobile Money, Orange Money ou carte bancaire. Des frais de service de 2 % s’ajoutent au montant des animaux ; le total est affiché avant tout paiement.',
        'La somme payée est conservée par Betailo (séquestre) et n’est pas versée immédiatement à l’éleveur. Elle lui est versée lorsque l’acheteur confirme la réception des animaux, ou automatiquement 48 heures après que l’éleveur a marqué la commande comme livrée, si l’acheteur n’a ni confirmé ni ouvert de litige.',
        'Ne payez jamais en dehors de la plateforme. Un paiement fait directement à l’éleveur n’est pas protégé, et une demande de paiement hors plateforme doit être signalée.'
      ]
    },
    {
      id: 'litiges',
      title: 'Litiges et remboursements',
      paragraphs: [
        'Une réservation payée ne peut pas être annulée directement : l’acheteur ouvre un litige depuis sa réservation, tant que le paiement est conservé par Betailo. Le versement à l’éleveur est alors suspendu.',
        'L’équipe Betailo examine le litige avec les deux parties et décide soit de verser la somme à l’éleveur, soit de rembourser l’acheteur par le moyen de paiement utilisé.'
      ]
    },
    {
      id: 'avis',
      title: 'Avis',
      paragraphs: [
        'Seul un acheteur dont la réservation est terminée peut noter une ferme. Les avis doivent être sincères et porter sur la transaction. Betailo peut retirer un avis injurieux, hors sujet ou manifestement faux.'
      ]
    },
    {
      id: 'interdits',
      title: 'Comportements interdits et signalements',
      paragraphs: [
        'Sont interdits notamment : les annonces fausses ou trompeuses, la vente d’animaux volés ou malades présentés comme sains, les demandes de paiement hors plateforme, l’usurpation d’identité, les propos injurieux ou discriminatoires sur la place de marché comme sur le forum.',
        'Chaque utilisateur peut signaler une annonce, un profil, un message du forum ou une réservation. Une annonce signalée trois fois en sept jours passe automatiquement en vérification manuelle. Betailo peut retirer un contenu et suspendre le compte concerné.'
      ]
    },
    {
      id: 'responsabilite',
      title: 'Responsabilité',
      paragraphs: [
        'Betailo n’élève, ne transporte et ne détient aucun animal. L’éleveur est seul responsable de la conformité, de la santé et de la livraison des animaux qu’il vend.',
        'Betailo met tout en œuvre pour que la plateforme soit disponible et que les paiements conservés soient versés ou remboursés conformément à ces conditions, sans pouvoir garantir un service ininterrompu.'
      ]
    },
    {
      id: 'donnees',
      title: 'Données personnelles',
      paragraphs: [
        'Betailo traite les données nécessaires au service : nom, adresse e-mail, téléphone, informations sur vos fermes, réservations et paiements. Elles ne sont pas vendues.',
        'Le téléphone de contact d’une ferme n’est montré qu’aux utilisateurs connectés. Le numéro Mobile Money n’est conservé que masqué : le numéro complet est transmis au seul prestataire de paiement, au moment du paiement.',
        'Vous pouvez demander l’accès, la correction ou la suppression de vos données auprès de l’équipe Betailo.'
      ]
    },
    {
      id: 'modifications',
      title: 'Modification des conditions',
      paragraphs: [
        'Betailo peut modifier ces conditions. La date de mise à jour figure en haut de cette page ; les changements importants sont annoncés aux utilisateurs. Continuer à utiliser la plateforme après un changement vaut acceptation de la nouvelle version.'
      ]
    },
    {
      id: 'droit',
      title: 'Droit applicable',
      paragraphs: [
        'Ces conditions sont régies par le droit camerounais. En cas de désaccord, les parties recherchent d’abord une solution amiable ; à défaut, les tribunaux compétents de Douala sont saisis.'
      ]
    }
  ],
  en: [
    {
      id: 'objet',
      title: 'About',
      paragraphs: [
        'Betailo is a marketplace that connects farmers and buyers to reserve animals from breeding cycles (chicks, piglets, kids…) or available for sale.',
        'The contract for the sale of the animals is made directly between the farmer and the buyer. Betailo is neither the farmer nor the seller: the platform provides the reservation tool, holds the payment until delivery and settles disputes.',
        'By creating an account, you accept these terms. If you do not accept them, do not use the platform.'
      ]
    },
    {
      id: 'compte',
      title: 'Your account',
      paragraphs: [
        'You agree to give accurate information and keep it up to date. An account is personal: you are responsible for what is done with it, and you keep your password secret.',
        'Betailo may suspend or close an account that breaks these terms, in particular for fraud or attempted scams.'
      ]
    },
    {
      id: 'eleveurs',
      title: 'Farms and farmers',
      paragraphs: [
        'Any account holder can register one or more farms. A farm only appears on the marketplace once the Betailo team has approved it, based on the proof provided.',
        'The farmer guarantees that their listings are accurate: species, number, age, price, dates, health and vaccination records. They hold the rights to the photos they publish and follow the applicable health and veterinary regulations.',
        'The farmer agrees to deliver animals that match the listing, within the announced dates.'
      ]
    },
    {
      id: 'reservations',
      title: 'Reservations',
      paragraphs: [
        'A reservation is for a quantity of animals from one listing. The unit price is fixed when you reserve, even if the farmer changes the price afterwards.',
        'An unpaid reservation puts the buyer on the cycle’s waiting list: it holds no places, the buyer can cancel it at any time, and it is cancelled automatically if the cycle ends before it is paid.',
        'Only payment holds places: a buyer who has paid is always served first.'
      ]
    },
    {
      id: 'paiement',
      title: 'Payment and escrow',
      paragraphs: [
        'Payment is made on the platform, by MTN Mobile Money, Orange Money or card. A 2% service fee is added to the price of the animals; the total is shown before any payment.',
        'The amount paid is held by Betailo (escrow) and is not paid to the farmer straight away. It is paid to them when the buyer confirms receipt of the animals, or automatically 48 hours after the farmer marks the order as delivered, if the buyer has neither confirmed nor opened a dispute.',
        'Never pay outside the platform. A payment made directly to the farmer is not protected, and any request to pay outside the platform should be reported.'
      ]
    },
    {
      id: 'litiges',
      title: 'Disputes and refunds',
      paragraphs: [
        'A paid reservation cannot be cancelled directly: the buyer opens a dispute from the reservation, as long as Betailo still holds the payment. Payment to the farmer is then put on hold.',
        'The Betailo team reviews the dispute with both parties and decides either to pay the farmer or to refund the buyer through the payment method used.'
      ]
    },
    {
      id: 'avis',
      title: 'Reviews',
      paragraphs: [
        'Only a buyer whose reservation is complete can rate a farm. Reviews must be honest and about the transaction. Betailo may remove a review that is abusive, off-topic or clearly false.'
      ]
    },
    {
      id: 'interdits',
      title: 'Prohibited behaviour and reports',
      paragraphs: [
        'Prohibited behaviour includes: false or misleading listings, selling stolen animals or sick animals presented as healthy, asking for payment outside the platform, impersonation, and abusive or discriminatory language on the marketplace and the forum.',
        'Anyone can report a listing, a profile, a forum post or a reservation. A listing reported three times in seven days is automatically sent to manual review. Betailo may remove content and suspend the account involved.'
      ]
    },
    {
      id: 'responsabilite',
      title: 'Liability',
      paragraphs: [
        'Betailo does not raise, transport or own any animal. The farmer alone is responsible for the conformity, health and delivery of the animals they sell.',
        'Betailo does its best to keep the platform available and to pay out or refund held payments as these terms describe, but cannot guarantee uninterrupted service.'
      ]
    },
    {
      id: 'donnees',
      title: 'Personal data',
      paragraphs: [
        'Betailo processes the data the service needs: name, email address, phone, information about your farms, reservations and payments. It is not sold.',
        'A farm’s contact phone is only shown to signed-in users. The Mobile Money number is only kept masked: the full number is passed only to the payment provider, at the time of payment.',
        'You can ask the Betailo team to access, correct or delete your data.'
      ]
    },
    {
      id: 'modifications',
      title: 'Changes to these terms',
      paragraphs: [
        'Betailo may change these terms. The date of the last update is shown at the top of this page, and significant changes are announced to users. Continuing to use the platform after a change means you accept the new version.'
      ]
    },
    {
      id: 'droit',
      title: 'Governing law',
      paragraphs: [
        'These terms are governed by Cameroonian law. In case of disagreement, the parties first seek an amicable solution; failing that, the competent courts of Douala have jurisdiction.'
      ]
    }
  ]
};
