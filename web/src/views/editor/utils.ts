import { FILE_RUNNERS } from '@/constants'

export function buildExecutionCommand(
  selectedFile: string,
  selectedEnvs: { plugin: string; version: string }[],
  baseDir: string
): string {
  const parts = selectedFile.split('/')
  const fileName = parts.pop() || selectedFile
  const dirPath = parts.join('/')
  const ext = fileName.split('.').pop()?.toLowerCase() || ''
  let runner = FILE_RUNNERS[ext] || ''

  const validEnvs = selectedEnvs.filter(e => e.plugin && e.version)
  let cmd = ''
  if (validEnvs.length > 0) {
    const specs = validEnvs.map(e => `${e.plugin}@${e.version}`).join(' ')
    if (!runner && !['sh', 'bash', 'bat', 'cmd', 'ps1'].includes(ext)) {
       const first = validEnvs[0]!.plugin
       runner = (first === 'node' ? 'node' : first)
    }
    cmd = `mise exec ${specs} -- ${runner ? `${runner} ${fileName}` : `./${fileName}`}`
  } else {
    cmd = runner ? `${runner} ${fileName}` : `./${fileName}`
  }

  return `cd ${baseDir}${dirPath ? '/' + dirPath : ''} && ${cmd}`
}
