/** 
 * PostCSS configuration for Tailwind CSS processing.
 * 
 * This file sets up the PostCSS plugins used throughout the project:
 *   - `postcss-import`   : Enables `@import` statements.
 *   - `tailwindcss`      : Generates Tailwind utilities based on `tailwind.config.js`.
 *   - `autoprefixer`     : Adds vendor prefixes for broader browser support.
 * 
 * The configuration is deliberately minimal – Tailwind's purge (content) 
 * handling lives in `tailwind.config.js`. In production we keep the same 
 * plugin set; any additional debugging plugins can be added locally without 
 * affecting the build pipeline.
 */

const isProduction = process.env.NODE_ENV === 'production';

let tailwindcss;
let autoprefixer;
let postcssImport;

try {
  // Core PostCSS plugins
  tailwindcss = require('tailwindcss');
  autoprefixer = require('autoprefixer');
  postcssImport = require('postcss-import');
} catch (err) {
  console.error('❌ Failed to load required PostCSS plugins:', err);
  // Exit early – without these plugins the build cannot succeed.
  process.exit(1);
}

/**
 * Export the PostCSS plugin map.
 * Using an object literal ensures deterministic ordering of plugins.
 */
module.exports = {
  plugins: {
    // Resolve `@import` rules before Tailwind processes the CSS.
    'postcss-import': postcssImport,

    // Tailwind CSS – reads `tailwind.config.js` for content paths, theme, etc.
    tailwindcss: tailwindcss,

    // Autoprefixer – adds necessary vendor prefixes based on browserslist.
    autoprefixer: autoprefixer,

    // In development we optionally enable `postcss-preset-env` for future CSS features.
    ...(isProduction
      ? {}
      : {
          'postcss-preset-env': {
            stage: 0,
            features: {
              'nesting-rules': true,
            },
          },
        }),
  },
};