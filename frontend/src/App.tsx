import { useEffect, useMemo, useState } from 'react'
import { Events } from '@wailsio/runtime'
import { GitService } from '../bindings/changeme'

type GitChange = {
  path: string
  originalPath: string
  indexStatus: string
  worktreeStatus: string
}

type SyncAction = 'fetch' | 'pull' | 'push'
type FolderChange = {
  folder: string
  files: GitChange[]
}

const ROOT_FOLDER = '.'

function getFolderFromPath(path: string): string {
  const normalized = path.replace(/\\/g, '/')
  const separatorIndex = normalized.lastIndexOf('/')
  if (separatorIndex <= 0) {
    return ROOT_FOLDER
  }
  return normalized.slice(0, separatorIndex)
}

function getSyncAction(statusOutput: string): SyncAction {
  const branchLine = statusOutput.split('\n')[0] ?? ''
  const behind = Number(branchLine.match(/behind (\d+)/)?.[1] ?? 0)
  const ahead = Number(branchLine.match(/ahead (\d+)/)?.[1] ?? 0)
  if (behind > 0) return 'pull'
  if (ahead > 0) return 'push'
  return 'fetch'
}

function App() {
  const [repositories, setRepositories] = useState<string[]>([])
  const [currentRepo, setCurrentRepo] = useState('')
  const [changes, setChanges] = useState<GitChange[]>([])
  const [selectedFolders, setSelectedFolders] = useState<Record<string, boolean>>({})
  const [message, setMessage] = useState('')
  const [description, setDescription] = useState('')
  const [action, setAction] = useState<SyncAction>('fetch')
  const [statusText, setStatusText] = useState('No repository selected.')
  const [busy, setBusy] = useState(false)

  const folderChanges = useMemo<FolderChange[]>(() => {
    const folders = new Map<string, GitChange[]>()
    for (const change of changes) {
      const folder = getFolderFromPath(change.path)
      const existing = folders.get(folder)
      if (existing) {
        existing.push(change)
      } else {
        folders.set(folder, [change])
      }
    }

    return Array.from(folders.entries())
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([folder, files]) => ({ folder, files }))
  }, [changes])

  const selectedPaths = useMemo(() => {
    const paths: string[] = []
    for (const folderChange of folderChanges) {
      if (!selectedFolders[folderChange.folder]) {
        continue
      }
      for (const file of folderChange.files) {
        paths.push(file.path)
      }
    }
    return paths
  }, [folderChanges, selectedFolders])

  const loadRepositories = () => {
    return GitService.ListRepositories().then((result) => {
      if (!result.success) {
        setStatusText(result.error || 'Failed to load repositories.')
        return
      }

      setRepositories(result.repositories || [])
      setCurrentRepo(result.current || result.repositories?.[0] || '')
    }).catch(() => {
      setStatusText('Failed to initialize repositories.')
    })
  }

  const loadRepoState = (repoPath: string) => {
    if (!repoPath) {
      setChanges([])
      setSelectedFolders({})
      setStatusText('No repository selected.')
      setAction('fetch')
      return Promise.resolve()
    }

    return Promise.all([
      GitService.Status(repoPath),
      GitService.ListChanges(repoPath),
    ]).then(([statusResult, changeResult]) => {
      if (statusResult.success) {
        setStatusText(statusResult.output || 'Ready.')
        setAction(getSyncAction(statusResult.output || ''))
      } else {
        setStatusText(statusResult.error || 'Failed to load status.')
      }

      if (!changeResult.success) {
        setChanges([])
        setSelectedFolders({})
        return
      }

      const nextChanges = (changeResult.changes || []) as GitChange[]
      setChanges(nextChanges)
      setSelectedFolders((previous) => {
        const next: Record<string, boolean> = {}
        for (const change of nextChanges) {
          const folder = getFolderFromPath(change.path)
          next[folder] = previous[folder] ?? false
        }
        return next
      })
    }).catch(() => {
      setStatusText('Failed to refresh repository state.')
    })
  }

  useEffect(() => {
    loadRepositories()
    Events.On('harbor:repositories-updated', () => {
      loadRepositories().catch(() => setStatusText('Failed to refresh repositories.'))
    })
  }, [])

  useEffect(() => {
    loadRepoState(currentRepo)
  }, [currentRepo])

  const runBusy = (operation: () => Promise<void>) => {
    setBusy(true)
    operation().finally(() => setBusy(false))
  }

  const handleSelectRepo = (repoPath: string) => {
    setCurrentRepo(repoPath)
    if (!repoPath) return

    GitService.SetCurrentRepository(repoPath).then((result) => {
      if (!result.success) {
        setStatusText(result.error || 'Failed setting repository.')
      }
    }).catch(() => {
      setStatusText('Failed setting repository.')
    })
  }

  const handleAddRepo = () => runBusy(() => {
    return GitService.SelectAndAddLocalRepository().then((result) => {
      if (!result.success) {
        setStatusText(result.error || 'Failed to add repository.')
        return
      }

      setRepositories(result.repositories || [])
      if (result.repository && !result.cancelled) {
        setCurrentRepo(result.repository)
      }
    }).catch(() => {
      setStatusText('Failed to add repository.')
    })
  })

  const handleSync = () => runBusy(() => {
    if (!currentRepo) {
      return Promise.resolve()
    }

    return GitService.SmartSync(currentRepo).then((result) => {
      setStatusText(result.success ? (result.output || `${result.action} complete`) : (result.error || `${result.action} failed`))
      return loadRepoState(currentRepo)
    }).catch(() => {
      setStatusText('Sync failed.')
    })
  })

  const handleCommit = () => runBusy(() => {
    if (!currentRepo || selectedPaths.length === 0) {
      return Promise.resolve()
    }

    return GitService.CommitSelected(currentRepo, selectedPaths, message, description).then((result) => {
      setStatusText(result.success ? (result.output || 'Commit created.') : (result.error || 'Commit failed.'))
      if (result.success) {
        setMessage('')
        setDescription('')
      }
      return loadRepoState(currentRepo)
    }).catch(() => {
      setStatusText('Commit failed.')
    })
  })

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-4xl flex-col gap-3 px-4 py-4 text-sm">
      <div className="flex flex-wrap gap-2 rounded-md border border-white/10 bg-[#161616] p-3">
        <select className="h-9 min-w-[18rem] flex-1 rounded-md border border-white/15 bg-white/5 px-2" value={currentRepo} onChange={(event) => handleSelectRepo(event.target.value)}>
          <option value="">Select repository</option>
          {repositories.map((repo) => <option key={repo} value={repo}>{repo}</option>)}
        </select>
        <button className="h-9 rounded-md border border-white/20 px-3" onClick={handleAddRepo} disabled={busy}>Add</button>
        <button className="h-9 rounded-md bg-blue-500 px-3 font-medium text-black disabled:opacity-60" onClick={handleSync} disabled={!currentRepo || busy}>
          {action === 'pull' ? 'Pull' : action === 'push' ? 'Push' : 'Fetch'}
        </button>
      </div>

      <div className="whitespace-pre-wrap rounded-md border border-white/10 bg-[#161616] px-3 py-2 text-xs text-gray-300">{statusText}</div>

      <div className="grid gap-3 md:grid-cols-[1.3fr_1fr]">
        <section className="rounded-md border border-white/10 bg-[#161616]">
          <div className="flex items-center justify-between border-b border-white/10 px-3 py-2">
            <h2>Changes</h2>
            <span>{selectedPaths.length} selected</span>
          </div>
          <div className="max-h-[62vh] overflow-auto">
            {folderChanges.length === 0 ? (
              <p className="p-3 text-xs text-gray-400">No local changes</p>
            ) : (
              <ul className="divide-y divide-white/10">
                {folderChanges.map((folderChange) => (
                  <li key={folderChange.folder} className="flex items-center gap-2 px-3 py-2">
                    <input
                      type="checkbox"
                      checked={!!selectedFolders[folderChange.folder]}
                      onChange={(event) => setSelectedFolders((previous) => ({ ...previous, [folderChange.folder]: event.target.checked }))}
                    />
                    <span className="min-w-[54px] text-xs text-gray-400">{folderChange.files.length} file{folderChange.files.length === 1 ? '' : 's'}</span>
                    <span className="truncate">{folderChange.folder}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </section>

        <section className="flex flex-col gap-2 rounded-md border border-white/10 bg-[#161616] p-3">
          <h2>Commit</h2>
          <input className="h-9 rounded-md border border-white/15 bg-white/5 px-2" value={message} onChange={(event) => setMessage(event.target.value)} placeholder="feat: added a feature" />
          <textarea className="min-h-28 rounded-md border border-white/15 bg-white/5 px-2 py-2" value={description} onChange={(event) => setDescription(event.target.value)} placeholder="Description" />
          <button className="h-9 rounded-md bg-green-500 px-3 font-medium text-black disabled:opacity-60" onClick={handleCommit} disabled={!currentRepo || !message.trim() || selectedPaths.length === 0 || busy}>
            Commit Selected
          </button>
        </section>
      </div>
    </main>
  )
}

export default App
