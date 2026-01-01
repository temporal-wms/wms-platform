// @ts-check
import {themes as prismThemes} from 'prism-react-renderer';

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'WMS Platform',
  tagline: 'Warehouse Management System - Enterprise Documentation',
  favicon: 'img/favicon.ico',

  url: 'https://wms-platform.example.com',
  baseUrl: '/',

  organizationName: 'wms-platform',
  projectName: 'wms-platform-docs',

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  markdown: {
    mermaid: true,
  },

  themes: ['@docusaurus/theme-mermaid'],

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: './sidebars.js',
          routeBasePath: '/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      image: 'img/wms-platform-social-card.jpg',
      navbar: {
        title: 'WMS Platform',
        logo: {
          alt: 'WMS Platform Logo',
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
            href: 'https://github.com/wms-platform/wms-platform',
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
        copyright: `Copyright © ${new Date().getFullYear()} WMS Platform. Built with Docusaurus.`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
        additionalLanguages: ['go', 'yaml', 'json', 'bash'],
      },
      mermaid: {
        theme: {light: 'neutral', dark: 'dark'},
      },
    }),
};

export default config;
