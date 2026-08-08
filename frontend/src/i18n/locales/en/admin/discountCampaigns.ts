export default {
  discountCampaigns: {
    title: 'Scheduled Discounts',
    description: 'Create time-limited Token multiplier discounts for standard groups.',
    create: 'New discount',
    edit: 'Edit discount',
    empty: 'No discount campaigns yet',
    emptyHint: 'Create a one-time or weekly campaign to display and apply reduced Token multipliers.',
    safetyTitle: 'Billing scope',
    safetyHint: 'Discounts apply only to Token billing for standard groups. Subscription groups and independently priced image, video, and Web Search usage are excluded.',
    fields: {
      name: 'Campaign name', enabled: 'Enable campaign', scheduleType: 'Schedule', timezone: 'Timezone',
      startsAt: 'Starts at', endsAt: 'Ends at', weekdays: 'Weekdays', allDay: 'All day',
      startTime: 'Daily start', endTime: 'Daily end', discountPercent: 'Payment percentage',
      minMultiplier: 'Minimum final multiplier', budgetCap: 'Discount budget cap'
    },
    scheduleTypes: { one_time: 'One-time', weekly: 'Weekly recurring' },
    weekdays: { sun: 'Sun', mon: 'Mon', tue: 'Tue', wed: 'Wed', thu: 'Thu', fri: 'Fri', sat: 'Sat' },
    hints: {
      discountPercent: 'For example, 90% turns 2x into 1.8x (equivalent to 10% off).',
      crossMidnight: 'An end time earlier than the start time continues into the next day.',
      minimum: 'Optional. The discounted multiplier will never fall below this value.',
      budget: 'Optional. New requests stop joining after the recorded savings reach this amount; in-flight requests retain their start-time price.',
      overlap: 'When campaigns overlap, the lowest effective multiplier wins.'
    },
    columns: { campaign: 'Campaign', schedule: 'Schedule', discount: 'Discount', budget: 'Savings / budget', status: 'Status', actions: 'Actions' },
    statuses: { active: 'Active', upcoming: 'Upcoming', ended: 'Ended', disabled: 'Disabled', budget_exhausted: 'Budget exhausted' },
    enabled: 'Enabled', disabled: 'Disabled', allDay: 'All day', noLimit: 'No limit',
    save: 'Verify and save', saving: 'Saving...', delete: 'Delete', deleteTitle: 'Delete discount campaign',
    deleteConfirm: 'Delete “{name}”? New requests will stop receiving this discount.',
    created: 'Discount campaign created.', updated: 'Discount campaign updated.', deleted: 'Discount campaign deleted.',
    errors: { load: 'Failed to load discount campaigns', save: 'Failed to save discount campaign', delete: 'Failed to delete discount campaign' }
  }
}
