import React from 'react'
import {createRoot} from 'react-dom/client'
import './main.css'
import {
    CommitSelected,
    ListChanges,
    ListRepositories,
    SelectAndAddLocalRepository,
    SetCurrentRepository,
    SmartSync,
    Status,
} from '../wailsjs/go/main/App'
import {EventsOn} from '../wailsjs/runtime/runtime'
import {main} from '../wailsjs/go/models'

type SyncAction = 'fetch' | 'pull' | 'push'

function getSyncAction(statusOutput: string): SyncAction {
    const branchLine = statusOutput.split('\n')[0] ?? ''
    const behind = Number(branchLine.match(/behind (\d+)/)?.[1] ?? 0)
    const ahead = Number(branchLine.match(/ahead (\d+)/)?.[1] ?? 0)

    if (behind > 0) return 'pull'
    if (ahead > 0) return 'push'
    return 'fetch'
}

function Main() {
    const [repositories, setRepositories] = React.useState<string[]>([])
    const [currentRepo, setCurrentRepo] = React.useState('')
    const [changes, setChanges] = React.useState<main.GitChange[]>([])
    const [selected, setSelected] = React.useState<Record<string, boolean>>({})
    const [message, setMessage] = React.useState('')
    const [description, setDescription] = React.useState('')
    const [action, setAction] = React.useState<SyncAction>('fetch')
    const [statusText, setStatusText] = React.useState('No repository selected.')
    const [busy, setBusy] = React.useState(false)

    const selectedPaths = React.useMemo(
        () => changes.filter((change) => selected[change.path]).map((change) => change.path),
        [changes, selected],
    )

    const loadRepositories = React.useCallback(async () => {
        const result = await ListRepositories()
        if (!result.success) {
            setStatusText(result.error || 'Failed to load repositories.')
            return
        }

        setRepositories(result.repositories || [])
        setCurrentRepo(result.current || result.repositories?.[0] || '')
    }, [])

    const loadRepoState = React.useCallback(async (repoPath: string) => {
        if (!repoPath) {
            setChanges([])
            setSelected({})
            setStatusText('No repository selected.')
            setAction('fetch')
            return
        }

        const [statusResult, changeResult] = await Promise.all([Status(repoPath), ListChanges(repoPath)])

        if (statusResult.success) {
            setStatusText(statusResult.output || 'Ready.')
            setAction(getSyncAction(statusResult.output || ''))
        } else {
            setStatusText(statusResult.error || 'Failed to load status.')
        }

        if (!changeResult.success) {
            setChanges([])
            setSelected({})
            return
        }

        setChanges(changeResult.changes || [])
        setSelected((previous) => {
            const next: Record<string, boolean> = {}
            for (const change of changeResult.changes || []) {
                next[change.path] = previous[change.path] ?? false
            }
            return next
        })
    }, [])

    React.useEffect(() => {
        loadRepositories().catch(() => setStatusText('Failed to initialize repositories.'))
        const off = EventsOn('harbor:repositories-updated', () => {
            loadRepositories().catch(() => setStatusText('Failed to refresh repositories.'))
        })
        return () => off()
    }, [loadRepositories])

    React.useEffect(() => {
        loadRepoState(currentRepo).catch(() => setStatusText('Failed to refresh repository state.'))
    }, [currentRepo, loadRepoState])

    const runBusy = async (operation: () => Promise<void>) => {
        setBusy(true)
        try {
            await operation()
        } finally {
            setBusy(false)
        }
    }

    const handleSelectRepo = async (repoPath: string) => {
        setCurrentRepo(repoPath)
        if (!repoPath) return

        const result = await SetCurrentRepository(repoPath)
        if (!result.success) {
            setStatusText(result.error || 'Failed setting repository.')
        }
    }

    const handleAddRepo = async () => runBusy(async () => {
        const result = await SelectAndAddLocalRepository()
        if (!result.success) {
            setStatusText(result.error || 'Failed to add repository.')
            return
        }

        setRepositories(result.repositories || [])
        if (result.repository && !result.cancelled) {
            setCurrentRepo(result.repository)
        }
    })

    const handleSync = async () => runBusy(async () => {
        if (!currentRepo) return
        const result = await SmartSync(currentRepo)
        setStatusText(result.success ? (result.output || `${result.action} complete`) : (result.error || `${result.action} failed`))
        await loadRepoState(currentRepo)
    })

    const handleCommit = async () => runBusy(async () => {
        if (!currentRepo || selectedPaths.length === 0) return

        const result = await CommitSelected(currentRepo, selectedPaths, message, description)
        setStatusText(result.success ? (result.output || 'Commit created.') : (result.error || 'Commit failed.'))
        if (result.success) {
            setMessage('')
            setDescription('')
        }
        await loadRepoState(currentRepo)
    })

    return (
        <main className="mx-auto flex min-h-screen w-full max-w-4xl flex-col gap-3 px-4 py-4 text-sm">
            <div className="flex flex-wrap items-center gap-2 rounded-md border border-white/10 p-3">
                <select
                    className="h-9 min-w-[18rem] flex-1 rounded-md border border-white/15 bg-white/5 px-2 outline-none"
                    value={currentRepo}
                    onChange={(event) => handleSelectRepo(event.target.value)}
                >
                    <option value="">Select repository</option>
                    {repositories.map((repo) => <option key={repo} value={repo}>{repo}</option>)}
                </select>
                <button className="h-9 rounded-md border border-white/20 px-3" onClick={handleAddRepo} disabled={busy}>Add</button>
                <button className="h-9 rounded-md bg-blue-500 px-3 font-medium text-black disabled:opacity-50" onClick={handleSync} disabled={!currentRepo || busy}>
                    {action === 'pull' ? 'Pull' : action === 'push' ? 'Push' : 'Fetch'}
                </button>
            </div>

            <div className="rounded-md border border-white/10 px-3 py-2 text-xs text-gray-300 whitespace-pre-wrap">{statusText}</div>

            <div className="grid gap-3 md:grid-cols-[1.25fr_1fr]">
                <section className="rounded-md border border-white/10">
                    <div className="flex items-center justify-between border-b border-white/10 px-3 py-2">
                        <h2 className="font-medium">Changes</h2>
                        <span className="text-xs text-gray-300">{selectedPaths.length} selected</span>
                    </div>
                    <div className="max-h-[50vh] overflow-auto">
                        {changes.length === 0 ? (
                            <p className="p-3 text-xs text-gray-300">No local changes</p>
                        ) : (
                            <ul className="divide-y divide-white/10">
                                {changes.map((change) => (
                                    <li key={`${change.path}-${change.indexStatus}-${change.worktreeStatus}`} className="flex items-center gap-2 px-3 py-2">
                                        <input
                                            type="checkbox"
                                            checked={!!selected[change.path]}
                                            onChange={(event) => setSelected((previous) => ({ ...previous, [change.path]: event.target.checked }))}
                                        />
                                        <span className="w-10 text-xs text-gray-300">{change.indexStatus}{change.worktreeStatus}</span>
                                        <span className="truncate">{change.path}</span>
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>
                </section>

                <section className="rounded-md border border-white/10 p-3">
                    <h2 className="mb-2 font-medium">Commit</h2>
                    <div className="flex flex-col gap-2">
                        <input
                            className="h-9 rounded-md border border-white/15 bg-white/5 px-2 outline-none"
                            value={message}
                            onChange={(event) => setMessage(event.target.value)}
                            placeholder="feat: added a feature"
                        />
                        <textarea
                            className="min-h-28 rounded-md border border-white/15 bg-white/5 px-2 py-2 outline-none"
                            value={description}
                            onChange={(event) => setDescription(event.target.value)}
                            placeholder="Description"
                        />
                        <button
                            className="h-9 rounded-md bg-green-500 px-3 font-medium text-black disabled:opacity-50"
                            onClick={handleCommit}
                            disabled={!currentRepo || !message.trim() || selectedPaths.length === 0 || busy}
                        >
                            Commit Selected
                        </button>
                    </div>
                </section>
            </div>
        </main>
    )
}

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <Main/>
    </React.StrictMode>
)
