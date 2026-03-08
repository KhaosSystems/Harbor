# Harbor — The Git Client for Games, Media, and Engineering
Harbor is a Git client built for game development, media production, and asset-heavy pipelines like architectural visualization and engineering (CAD).

Git is the standard for software development, but many studios still rely on proprietary systems like Perforce Helix Core (P4) for production work with large binary assets.

Today, Git has largely caught up. With Git LFS 2.0, file locking, and sparse checkout, it supports the workflows needed for large projects.

The problem now is tooling.

Most Git clients are built for software developers. They don't expose these features well, and they are difficult to adapt to production pipelines, automation, and heavy preflight workflows used in games and media.

Harbor is an attempt to fix that. It's a Git client designed for artists, designers, and technical teams, with better support for asset pipelines, hooks, and heavy automation. 

# Wails (TODO: Rewrite)
## About

This is the official Wails React-TS template.

You can configure the project by editing `wails.json`. More information about the project settings can be found
here: https://wails.io/docs/reference/project-config

## Live Development

To run in live development mode, run `wails dev` in the project directory. This will run a Vite development
server that will provide very fast hot reload of your frontend changes. If you want to develop in a browser
and have access to your Go methods, there is also a dev server that runs on http://localhost:34115. Connect
to this in your browser, and you can call your Go code from devtools.

## Building

To build a redistributable, production mode package, use `wails build`.
