# WMS Labs Ecosystem Documentation

This website is built using [Docusaurus 2](https://docusaurus.io/), a modern static website generator.

## Prerequisites

- Node.js version 20.0 or above:
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

Documentation is automatically deployed to GitHub Pages: **https://temporal-wms.github.io/wms-platform/**

### Automatic Deployment

Changes pushed to the `main` branch in `ecosystem-documentation/` automatically trigger deployment via GitHub Actions.

- **Workflow:** `.github/workflows/deploy-docs.yml`
- **Build time:** ~2-3 minutes
- **Monitor:** https://github.com/temporal-wms/wms-platform/actions

### Manual Deployment

You can also deploy manually if needed:

**Mac/Linux:**
```bash
GIT_USER=<Your GitHub username> npm run deploy
```

**Windows:**
```bash
cmd /C "set "GIT_USER=<Your GitHub username>" && npm run deploy"
```

### Configuration

- **URL:** https://temporal-wms.github.io
- **Base URL:** /wms-platform/
- **Deployment Branch:** gh-pages
- **Build Tool:** Docusaurus 3.6.0

### Testing Before Deployment

Always test the production build locally:

```bash
# Build
npm run build

# Serve (with production baseUrl)
npm run serve
```

Opens http://localhost:3000/wms-platform/

### Troubleshooting

For detailed deployment instructions, troubleshooting, and rollback procedures, see [DEPLOYMENT.md](./DEPLOYMENT.md).

## Project Structure

- `/docs`: Documentation source files (Markdown/MDX)
- `/src`: React components and pages
- `/static`: Static assets (images, etc.)
- `docusaurus.config.js`: Site configuration
