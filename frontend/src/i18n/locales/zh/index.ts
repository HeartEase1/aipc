import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import batchImage from './batchImage'
import benefits from './benefits'
import admin from './admin'
import misc from './misc'
import { zh as balanceMarketing } from '../balanceMarketing'

export default {
  balanceMarketing,
  ...landing,
  ...common,
  ...dashboard,
  ...channelMonitorV2,
  ...batchImage,
  ...benefits,
  admin,
  ...misc,
}
