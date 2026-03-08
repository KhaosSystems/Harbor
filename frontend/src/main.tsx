import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './main.css'

/** TODO: Merge with Matter */
function MtDropdown({ children, className }) {
    return (
        <div className={`mt-surface-panel flex flex-row px-2 py-2 justify-center items-center rounded-md justify-start ${className}`}>
            {children}
        </div>
    );
}

function MtButton({ children, className }) {
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
    return (
        <div className="p-2 grid grid-cols-[300px_1fr] grid-rows-[auto_1fr] gap-2 h-screen">
            { /* Logo and Repo select */ }
            <div className="flex flex-row gap-2 items-center">
                <img src="/harbor-logo.svg" className="ml-1.5 w-7 h-7"/>
                <MtDropdown className="flex-1" variant="panel" size="large">Dropdown</MtDropdown>
            </div>

            { /* Branch Select, Actions and Window Controls */ }
            <div className="flex flex-row gap-2">
                <MtDropdown variant="panel" size="large">Branch</MtDropdown>
                <MtButton variant="panel" size="large">0</MtButton>
                <MtButton variant="panel" size="large">1</MtButton>
                <MtButton variant="panel" size="large">2</MtButton>
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
    );
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
    <React.StrictMode>
        <App2/>
    </React.StrictMode>,
)
