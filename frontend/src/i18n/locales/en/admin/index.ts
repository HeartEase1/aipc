import overview from './overview'
import channels from './channels'
import accounts from './accounts'
import resources from './resources'
import ops from './ops'
import settings from './settings'
import audit from './audit'
import promptAudit from './promptAudit'
import plugins from './plugins'

import benefitGrants from './benefitGrants'
import discountCampaigns from './discountCampaigns'
import groupModelCompatibility from './groupModelCompatibility'

export default {
  groupModelCompatibility,
  ...benefitGrants,
  ...discountCampaigns,
  ...overview,
  ...channels,
  ...accounts,
  ...resources,
  ...ops,
  ...settings,
  ...audit,
  ...promptAudit,
  ...plugins,
}
