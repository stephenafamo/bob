// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const { Joi } = require("@docusaurus/utils-validation");
const lightCodeTheme = require("prism-react-renderer/themes/github");
const darkCodeTheme = require("prism-react-renderer/themes/dracula");

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: "Bob",
  tagline: "Go SQL Access Toolkit",
  url: "https://bob.stephenafamo.com",
  baseUrl: "/",
  onBrokenLinks: "throw",
  onBrokenMarkdownLinks: "warn",
  favicon: "img/favicon.ico",

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: "stephenafamo", // Usually your GitHub org/user name.
  projectName: "bob", // Usually your repo name.

  webpack: {
    jsLoader: (isServer) => ({
      loader: require.resolve("esbuild-loader"),
      options: {
        loader: "tsx",
        format: isServer ? "cjs" : undefined,
        target: isServer ? "node12" : "es2017",
      },
    }),
  },

  // Even if you don't use internalization, you can use this field to set useful
  // metadata like html lang. For example, if your site is Chinese, you may want
  // to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },

  presets: [
    [
      "classic",
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          path: "./docs",
          sidebarPath: require.resolve("./sidebars.js"),
          editUrl: "https://github.com/stephenafamo/bob/tree/main/website/",
        },
        theme: {
          customCss: require.resolve("./src/css/custom.css"),
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      navbar: {
        title: "Bob - Go SQL Access Toolkit",
        items: [
          {
            type: "dropdown",
            label: "Query Building",
            position: "left",
            items: [
              {
                label: "Introduction",
                to: "docs/query-builder/intro",
              },
              {
                label: "PostgreSQL",
                to: "docs/query-builder/psql/how-to-use",
              },
              {
                label: "MySQL",
                to: "docs/query-builder/mysql/how-to-use",
              },
              {
                label: "SQLite",
                to: "docs/query-builder/sqlite/how-to-use",
              },
            ],
          },
          {
            type: "dropdown",
            label: "Code Generation",
            position: "left",
            items: [
              {
                label: "Introduction",
                to: "docs/code-generation/intro",
              },
              {
                label: "PostgreSQL",
                to: "docs/code-generation/psql",
              },
              {
                label: "MySQL",
                to: "docs/code-generation/mysql",
              },
              {
                label: "SQLite",
                to: "docs/code-generation/sqlite",
              },
              {
                label: "SQL Files",
                to: "docs/code-generation/sql",
              },
            ],
          },
          {
            type: "dropdown",
            label: "VS Others",
            position: "left",
            items: [
              {
                label: "GORM",
                to: "vs/gorm",
              },
              {
                label: "Ent",
                to: "vs/ent",
              },
              {
                label: "SQLBoiler",
                to: "vs/sqlboiler",
              },
              {
                label: "Jet",
                to: "vs/jet",
              },
            ],
          },
          {
            href: "https://github.com/stephenafamo/bob",
            label: "GitHub",
            position: "right",
          },
          {
            href: "https://pkg.go.dev/github.com/stephenafamo/bob",
            label: "Reference",
            position: "right",
          },
        ],
      },
      footer: {
        style: "dark",
        copyright: `Copyright © ${new Date().getFullYear()} Stephen Afam-Osemene. Built with Docusaurus.`,
      },
      prism: {
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
      },
    }),

  plugins: [
    [
      "@docusaurus/plugin-content-docs",
      {
        id: "comparisons",
        path: "./vs",
        routeBasePath: "vs",
        sidebarPath: require.resolve("./sidebars.js"),
        editUrl: "https://github.com/stephenafamo/bob/tree/main/website/",
      },
    ],
    [
      function pluginAnalytics(
        /** @type {import('@docusaurus/types').LoadContext} */
        context,
        /** @type {{websiteID: string}} */
        options,
      ) {
        const url = new URL(context.siteConfig.url);
        const isProd = process.env.NODE_ENV === "production";

        /** @type {import('Joi').ObjectSchema<{websiteID: string}>} */
        const pluginOptionsSchema = Joi.object({
          websiteID: Joi.string().guid().required(),
        });

        const { value: cleanOptions, error } = pluginOptionsSchema.validate(
          options,
          { allowUnknown: true },
        );
        if (error?.details) {
          throw error;
        }

        const { websiteID } = cleanOptions;

        return {
          name: "docusaurus-plugin-analytics",
          async contentLoaded({ actions }) {
            actions.setGlobalData(options);
          },
          injectHtmlTags() {
            let baseUrl = `http://swishink.test`;
            if (isProd) {
              baseUrl = `https://swish.ink`;
            }

            return {
              postBodyTags: [
                `<script async defer data-api-url="${baseUrl}/" data-hostname="${websiteID}.${url.hostname}" data-collect-dnt="true" src="${baseUrl}/assets/js/latest.js"></script>`,
                `<noscript><img src="${baseUrl}/simple.gif?collect-dnt=true&hostname=${websiteID}.${url.hostname}" alt="" referrerpolicy="no-referrer-when-downgrade"/></noscript>`,
              ],
            };
          },
        };
      },
      /** @type {{websiteID: string}} */
      {
        websiteID: "e84c4b3f-1915-5441-b601-d1b85dce7329", // Required
      },
    ],
    async function tailwindPlugin(_context, _options) {
      return {
        name: "docusaurus-tailwindcss",
        configurePostCss(postcssOptions) {
          // Appends TailwindCSS and AutoPrefixer.
          postcssOptions.plugins.push(require("tailwindcss"));
          postcssOptions.plugins.push(require("autoprefixer"));
          return postcssOptions;
        },
      };
    },
  ],
};

module.exports = config;
