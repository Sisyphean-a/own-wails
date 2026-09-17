export type SyncActivity =
  | ''
  | 'switching_profile'
  | 'loading_snapshots'
  | 'uploading'
  | 'downloading'
  | 'checking_conflicts'
  | 'applying_snapshot'

export function isBusyActivity(activity: SyncActivity): boolean {
  return activity !== ''
}

export function describeSyncActivity(activity: SyncActivity): string {
  switch (activity) {
    case 'switching_profile':
      return '正在切换配置集...'
    case 'loading_snapshots':
      return '正在加载最新快照...'
    case 'uploading':
      return '正在上传文件到云端...'
    case 'downloading':
      return '正在拉取云端数据并同步到本地...'
    case 'checking_conflicts':
      return '正在检查本地冲突...'
    case 'applying_snapshot':
      return '正在写入文件到本地...'
    default:
      return ''
  }
}

export function quickUploadButtonLabel(activity: SyncActivity): string {
  return activity === 'uploading' ? '正在上传配置...' : '一键上传配置'
}

export function quickDownloadButtonLabel(activity: SyncActivity): string {
  if (activity === 'applying_snapshot') {
    return '正在执行覆盖...'
  }
  return activity === 'downloading' ? '正在下载更新...' : '一键下载更新'
}

export function advancedUploadButtonLabel(activity: SyncActivity): string {
  return activity === 'uploading' ? '正在上传选中条目...' : '上传选中条目'
}

export function advancedDownloadButtonLabel(activity: SyncActivity): string {
  switch (activity) {
    case 'checking_conflicts':
      return '正在预检冲突...'
    case 'applying_snapshot':
      return '正在应用快照...'
    default:
      return '预检冲突并应用快照'
  }
}
