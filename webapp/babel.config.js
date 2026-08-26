// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

const config = {

    // Top level rather than inside preset-env: babel-plugin-polyfill-corejs3 reads the
    // targets from here, and preset-env inherits them. Nested targets are invisible to
    // the polyfill plugin, which then injects nothing at all.
    targets: {
        chrome: 66,
        firefox: 60,
        edge: 42,
        safari: 12,
    },
    presets: [
        ['@babel/preset-env', {
            modules: false,
            debug: false,
            shippedProposals: true,
        }],

        // The plugin's webpack config externalizes `react` to the host webapp's
        // global React instance; the default "automatic" runtime would instead
        // bundle this project's own react/jsx-(dev-)runtime copy, which then
        // holds internal state that doesn't match the host's React instance.
        ['@babel/preset-react', {runtime: 'classic'}],

        // Babel 8 removed `allExtensions`/`isTSX`; the preset now decides whether to
        // parse JSX from the file extension, which is what this project wants.
        '@babel/preset-typescript',
    ],
    plugins: [

        // Babel 8 dropped preset-env's `useBuiltIns`/`corejs` options in favour of this
        // plugin. `usage-global` injects the same per-file core-js imports the old
        // `useBuiltIns: 'usage'` did; `version` tracks the core-js dependency.
        ['polyfill-corejs3', {
            method: 'usage-global',
            version: '3.50',
        }],
    ],
};

// Jest needs module transformation
config.env = {
    test: {
        presets: config.presets,
        plugins: config.plugins,
    },
};
config.env.test.presets[0][1].modules = 'auto';

module.exports = config;
