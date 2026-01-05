# WMS Labs Ecosystem Documentation

This website is built using [Docusaurus 2](https://docusaurus.io/), a modern static website generator.

## Prerequisites

- Node.js version 18.0 or above:
  - Check your version with `node -v`
  - [Download Node.js](https://nodejs.org/)

## Installation

Install the dependencies:

```bash
npm install
```

## Local Development

Starts the development server.

```bash
npm start
```

This command starts a local development server and opens up a browser window. Most changes are reflected live without having to restart the server.

## Build

Build the website for production.

```bash
npm run build
```

This command generates static content into the `build` directory and can be served using any static contents hosting service.

## Deployment

Deploy the website to GitHub Pages (or other hosting providers).

```bash
cmd /C "set "GIT_USER=<Your GitHub username>" && npm run deploy"
```

OR (on Mac/Linux):

```bash
GIT_USER=<Your GitHub username> npm run deploy
```

## Project Structure

- `/docs`: Documentation source files (Markdown/MDX)
- `/src`: React components and pages
- `/static`: Static assets (images, etc.)
- `docusaurus.config.js`: Site configuration
