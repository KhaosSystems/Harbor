import React from 'react'
import ReactDOM from 'react-dom/client'
import type { CSSProperties, ReactNode } from 'react'
import { Window } from '@wailsio/runtime'
import App from './App'
import './main.css'

const dragRegionStyle = { '--wails-draggable': 'drag' } as CSSProperties
const noDragStyle = { '--wails-draggable': 'no-drag' } as CSSProperties

type MtProps = {
    children: ReactNode
    className?: string
}

/** TODO: Merge with Matter */
function MtDropdown({ children, className = '' }: MtProps) {
    return (
        <div className={`mt-surface-panel flex flex-row px-2 py-2 justify-center items-center rounded-md justify-start ${className}`}>
            {children}
        </div>
    );
}

function MtButton({ children, className = '' }: MtProps) {
    return (
        <div className={`mt-surface-panel aspect-square flex flex-row px-2 py-2 justify-items-center items-center rounded-md justify-start ${className}`}>
            {children}
        </div>
    );
}

function Changes() {
    return (
      <div className="flex-1 bg-surface-subtle p-2">
        <ul>
            <li>File 1</li>
            <li>File 2</li>
            <li>File 3</li>
        </ul>
      </div>
    );
}

function CommitToolbar() {
    return (
      <div className="bg-surface-panel p-2">
          <input className="h-16 w-full rounded-md bg-surface-base px-2" placeholder="feat: added a feature" />
      </div>
    )
}

function App2() {
    const handleWindowAction = (operation: () => Promise<void>) => {
        operation().catch(() => undefined)
    }

    return (
        <div className="h-screen bg-surface-base text-text-primary">
            <div className="p-2 h-full grid grid-cols-[300px_1fr] grid-rows-[auto_1fr] gap-2 h-[calc(100vh-2.5rem)]">
            { /* Logo and Repo select */ }
            <div className="flex flex-row gap-2 items-center">
                <img src="/harbor-logo.svg" className="ml-1.5 w-7 h-7"/>
                <MtDropdown className="flex-1">Dropdown</MtDropdown>
            </div>

            { /* Branch Select, Actions and Window Controls */ }
            <div className="flex flex-row gap-2 items-center" style={dragRegionStyle}>
                <MtDropdown>Branch</MtDropdown>
                <MtButton>0</MtButton>
                <MtButton>1</MtButton>
                <MtButton>2</MtButton>
                <div className="ml-auto flex items-center gap-1" style={noDragStyle}>
                    <button
                        type="button"
                        className="h-8 w-8 rounded text-xs hover:bg-input-base-background-hover"
                        onClick={() => handleWindowAction(Window.Minimise)}
                    >
                        —
                    </button>
                    <button
                        type="button"
                        className="h-8 w-8 rounded text-xs hover:bg-input-base-background-hover"
                        onClick={() => handleWindowAction(Window.ToggleMaximise)}
                    >
                        ▢
                    </button>
                    <button
                        type="button"
                        className="h-8 w-8 rounded text-xs hover:bg-status-danger/25"
                        onClick={() => handleWindowAction(Window.Close)}
                    >
                        ✕
                    </button>
                </div>
            </div>

            { /* Explorer */ }
            <div className="mt-surface-panel">
                { /* Files / History switch */ }
                <div className="flex flex-row gap-2 p-2 border-b border-border-subtle">
                    <div className="flex-1 justify-items-center items-center rounded-md px-2 py-0.5 bg-surface-popover">
                        Changes
                    </div>
                    <div className="flex-1 justify-items-center items-center rounded-md px-2 py-0.5">
                        History
                    </div>
                </div>

                { /* File Tree */ }
                File Tree
            </div>

            { /* Content */ }
            <div className="flex flex-col mt-surface-panel">
                { /* Changes Toolbar */ }
                <div className="flex flex-row gap-2 p-2 border-b border-border-subtle">
                    Filter, Sort, View
                </div>

                { /**/ }
                <Changes/>

                { /* Commit Toolbar */ }
                <CommitToolbar />
            </div>
            </div>
        </div>
    );
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
    <React.StrictMode>
        <App2/>
    </React.StrictMode>,
)
