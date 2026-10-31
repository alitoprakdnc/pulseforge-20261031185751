module.exports = {
  root: true,
  parser: '@typescript-eslint/parser',
  parserOptions: {
    ecmaVersion: 2021,
    sourceType: 'module',
    ecmaFeatures: { jsx: true },
    project: ['./tsconfig.json'],
  },
  env: {
    browser: true,
    node: true,
    es2021: true,
  },
  plugins: [
    'react',
    '@typescript-eslint',
    'jsx-a11y',
    'react-hooks',
    'prettier',
    'import',
  ],
  extends: [
    'eslint:recommended',
    'plugin:react/recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:jsx-a11y/recommended',
    'plugin:react-hooks/recommended',
    'plugin:prettier/recommended',
  ],
  settings: {
    react: {
      version: 'detect',
    },
    'import/resolver': {
      node: {
        extensions: ['.js', '.jsx', '.ts', '.tsx'],
      },
    },
  },
  rules: {
    // Prettier integration
    'prettier/prettier': 'error',

    // General best‑practice
    'no-console': ['warn', { allow: ['error'] }],
    'no-debugger': 'error',
    'no-var': 'error',
    'prefer-const': 'error',
    'no-multi-spaces': 'error',
    'no-trailing-spaces': 'error',
    'eol-last': ['error', 'always'],

    // TypeScript specific
    '@typescript-eslint/explicit-function-return-type': 'off',
    '@typescript-eslint/no-unused-vars': [
      'error',
      { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
    ],
    '@typescript-eslint/no-explicit-any': 'warn',
    '@typescript-eslint/consistent-type-imports': 'error',
    '@typescript-eslint/no-var-requires': 'off',
    '@typescript-eslint/strict-boolean-expressions': 'error',
    '@typescript-eslint/prefer-nullish-coalescing': 'error',
    '@typescript-eslint/prefer-optional-chain': 'error',
    '@typescript-eslint/array-type': ['error', { default: 'generic' }],

    // React
    'react/prop-types': 'off',
    'react/react-in-jsx-scope': 'off',
    'react/jsx-uses-react': 'off',
    'react/jsx-boolean-value': ['error', 'never'],
    'react/jsx-curly-spacing': ['error', { when: 'always', children: true }],
    'react/jsx-no-duplicate-props': ['error', { ignoreCase: true }],
    'react/jsx-fragments': ['error', 'syntax'],
    'react/function-component-definition': [
      'error',
      { namedComponents: 'arrow-function', unnamedComponents: 'arrow-function' },
    ],
    'react/self-closing-comp': ['error', { component: true, html: true }],
    'react/jsx-tag-spacing': ['error', { beforeSelfClosing: 'always' }],
    'react/jsx-sort-props': [
      'error',
      {
        callbacksLast: true,
        shorthandFirst: true,
        noSortAlphabetically: false,
        reservedFirst: true,
      },
    ],

    // Accessibility
    'jsx-a11y/anchor-is-valid': [
      'error',
      {
        components: ['Link'],
        specialLink: ['hrefLeft', 'hrefRight'],
        aspects: ['invalidHref', 'preferButton'],
      },
    ],
    'jsx-a11y/no-autofocus': 'warn',
    'jsx-a11y/heading-has-content': 'error',
    'jsx-a11y/label-has-associated-control': 'error',

    // React Hooks
    'react-hooks/rules-of-hooks': 'error',
    'react-hooks/exhaustive-deps': 'warn',

    // Import ordering
    'import/order': [
      'error',
      {
        groups: [['builtin', 'external'], 'internal', ['parent', 'sibling', 'index']],
        'newlines-between': 'always',
        alphabetize: { order: 'asc', caseInsensitive: true },
      },
    ],
    'import/no-unresolved': 'error',
    'import/extensions': [
      'error',
      'ignorePackages',
      {
        js: 'never',
        jsx: 'never',
        ts: 'never',
        tsx: 'never',
      },
    ],

    // Test files (Jest)
    'jest/no-disabled-tests': 'warn',
    'jest/no-focused-tests': 'error',
    'jest/no-identical-title': 'error',
  },
  overrides: [
    {
      files: ['**/*.test.{ts,tsx}', '**/*.spec.{ts,tsx}'],
      env: { jest: true },
      plugins: ['jest'],
      extends: ['plugin:jest/recommended'],
      rules: {
        '@typescript-eslint/no-explicit-any': 'off',
      },
    },
  ],
};