/** The "pay now vs. reserve without paying" choice appears three times in
 * the design (Checkout, its mobile variant, and the condensed booking box
 * on a listing) with identical wording rules — kept in one place so the
 * three surfaces can't drift. The `priority` a reservation actually gets
 * is decided by the backend and returned on the reservation; this only
 * decides what the form says and shows next. Returns translation keys,
 * not literal text — see `core/i18n`. */
export interface PriorityChoiceCopy {
  readonly titleKey: string;
  readonly noticeTitleKey: string;
  readonly noticeBodyKey: string;
  readonly showsPaymentMethod: boolean;
}

export function priorityChoiceCopy(payNow: boolean): PriorityChoiceCopy {
  if (payNow) {
    return {
      titleKey: 'priorityChoice.payNowTitle',
      noticeTitleKey: 'priorityChoice.payNowNoticeTitle',
      noticeBodyKey: 'priorityChoice.payNowNoticeBody',
      showsPaymentMethod: true
    };
  }
  return {
    titleKey: 'priorityChoice.laterTitle',
    noticeTitleKey: 'priorityChoice.laterNoticeTitle',
    noticeBodyKey: 'priorityChoice.laterNoticeBody',
    showsPaymentMethod: false
  };
}
