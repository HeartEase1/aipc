export default {
  benefitGrants: {
    title: 'Welfare & Compensation Grants',
    description: 'Add available balance to users with a locked preview and step-up verification.',
    tabs: { create: 'New grant', history: 'Grant history' },
    sections: { rules: 'Grant rules', protection: 'Amount safeguards', notification: 'Arrival notification' },
    fields: {
      type: 'Grant type', mode: 'Grant method', audience: 'Recipients', fixedAmount: 'Fixed amount per user',
      percentage: 'Percentage of spending in the last 24 hours', minAmount: 'Minimum grant', perUserCap: 'Per-user cap',
      totalBudgetCap: 'Batch budget cap', reason: 'Reason', notificationTitle: 'Popup title',
      notificationContent: 'Popup content (Markdown supported)', user: 'User'
    },
    types: { welfare: 'Welfare', compensation: 'Compensation' },
    modes: { fixed: 'Fixed amount', percentage_24h: '24-hour spending percentage' },
    audiences: { all: 'All eligible users', selected: 'Selected users' },
    audienceHints: {
      all: 'Grant to every active, non-deleted ordinary user.',
      selected: 'Search and select up to 500 eligible ordinary users.'
    },
    searchUsers: 'Search by email or username',
    selectedCount: '{count} selected',
    summary: { title: 'Current rules' },
    safety: {
      title: 'Excluded from recharge and affiliate totals',
      content: 'This only adds available balance. It does not increase recharge totals, trigger affiliate rebates, or alter billing, pricing, or subscription quotas.'
    },
    preview: 'Preview grant',
    previewing: 'Generating preview...',
    previewNotification: 'Preview arrival popup',
    confirmTitle: 'Confirm grant',
    confirmAcknowledgement: 'I have reviewed the recipients, calculation rules, and estimated total, and confirm immediate execution.',
    execute: 'Verify and grant',
    executing: 'Submitting...',
    submitted: 'The batch was submitted and will continue processing in the background.',
    retryFailed: 'Retry failed items',
    retrySubmitted: 'Failed items were resubmitted.',
    detailTitle: 'Grant details',
    emptyHistory: 'No grant batches yet',
    allStatuses: 'All statuses',
    overBudget: 'The estimated total exceeds the batch budget cap and cannot be executed.',
    columns: { batch: 'Batch', progress: 'Succeeded / expected', amount: 'Amount', created: 'Created' },
    metrics: {
      recipients: 'Expected recipients', skipped: 'Skipped', baseCost: '24-hour spending base',
      totalAmount: 'Estimated total', average: 'Average amount', maximum: 'Largest grant', amount: 'Grant amount',
      succeeded: 'Succeeded', failed: 'Failed', distributed: 'Distributed', window: 'Locked spending window'
    },
    statuses: {
      draft: 'Awaiting confirmation', pending: 'Pending', processing: 'Processing', completed: 'Completed',
      partially_failed: 'Partially failed', failed: 'Failed', expired: 'Expired'
    },
    itemStatuses: {
      pending: 'Pending', succeeded: 'Received', failed: 'Failed', skipped_ineligible: 'Ineligible and skipped'
    },
    defaults: {
      title: 'You received a balance grant',
      previewReason: 'Example grant reason',
      content: 'You received **{\'{{amount}}\'}** in balance.\n\nReason: {\'{{reason}}\'}\n\nCurrent balance: {\'{{balance}}\'}\n\nThank you for using {\'{{site_name}}\'}.'
    },
    errors: {
      preview: 'Failed to generate grant preview', execute: 'Failed to submit grant', load: 'Failed to load grant history',
      retry: 'Failed to retry grant items', export: 'Failed to export grant details', search: 'Failed to search users',
      selectedLimit: 'No more than 500 users can be selected manually'
    }
  }
}
