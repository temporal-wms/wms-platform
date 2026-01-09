// @ts-check
import { themes as prismThemes } from 'prism-react-renderer';

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'WMS Labs',
  tagline: 'Warehouse Management System - Enterprise Documentation',
  favicon: 'img/favicon.ico',

  url: 'https://temporal-wms.github.io',
  baseUrl: '/wms-platform/',

  organizationName: 'temporal-wms',
  projectName: 'wms-platform',
  deploymentBranch: 'gh-pages',
  trailingSlash: false,

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  markdown: {
    mermaid: true,
  },

  themes: [
    '@docusaurus/theme-mermaid',
    'docusaurus-theme-openapi-docs',
  ],

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: './sidebars.js',
          routeBasePath: '/',
          docItemComponent: '@theme/ApiItem',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      }),
    ],
  ],

  plugins: [
    [
      'docusaurus-plugin-openapi-docs',
      {
        id: 'api',
        docsPluginId: 'classic',
        config: {
          wmsLabs: {
            specPath: 'docs/api/specs/openapi/order-service.yaml',
            outputDir: 'docs/api/order-service',
            sidebarOptions: {
              groupPathsBy: 'tag',
            },
          },
        },
      },
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      image: 'img/wms-platform-social-card.jpg',
      navbar: {
        title: 'WMS Labs',
        logo: {
          alt: 'WMS Labs Logo',
          src: 'img/logo.svg',
        },
        items: [
          {
            type: 'docSidebar',
            sidebarId: 'docsSidebar',
            position: 'left',
            label: 'Documentation',
          },
          {
            href: 'https://github.com/temporal-wms/wms-platform',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Documentation',
            items: [
              {
                label: 'Architecture',
                to: '/architecture/overview',
              },
              {
                label: 'Domain-Driven Design',
                to: '/domain-driven-design/overview',
              },
              {
                label: 'Services',
                to: '/services/order-service',
              },
            ],
          },
          {
            title: 'Resources',
            items: [
              {
                label: 'API Reference',
                to: '/api/rest-api',
              },
              {
                label: 'Infrastructure',
                to: '/infrastructure/overview',
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} WMS Labs. Built with Docusaurus.`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
        additionalLanguages: ['go', 'yaml', 'json', 'bash'],
      },
      mermaid: {
        theme: { light: 'neutral', dark: 'dark' },
      },
    }),
};

export default config;
