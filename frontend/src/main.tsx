import React from 'react'
import {createRoot} from 'react-dom/client'
import './main.css'
import {Greet} from '../wailsjs/go/main/App'

function Main() {
    const [resultText, setResultText] = React.useState('Please enter your name below.')
    const [name, setName] = React.useState('')

    function greet() {
        Greet(name).then(setResultText)
    }

    return (
        <main className="mx-auto flex min-h-screen w-full max-w-xl flex-col items-center justify-center gap-6 px-6">
            <h1 className="text-3xl font-semibold">Harbor</h1>
            <p className="text-sm text-gray-300">{resultText}</p>
            <div className="flex w-full gap-3">
                <input
                    id="name"
                    className="h-10 flex-1 rounded-md border border-white/15 bg-white/5 px-3 outline-none focus:border-white/30"
                    onChange={(event) => setName(event.target.value)}
                    autoComplete="off"
                    name="input"
                    type="text"
                    placeholder="Enter your name"
                />
                <button className="h-10 rounded-md bg-blue-500 px-4 font-medium text-black" onClick={greet}>Greet</button>
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
