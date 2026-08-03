// @ts-check
const eslint = require('@eslint/js');
const tseslint = require('typescript-eslint');
const angular = require('angular-eslint');
const importPlugin = require('eslint-plugin-import');
const deprecatePlugin = require('eslint-plugin-deprecate');
const prettierConfig = require('eslint-config-prettier');
module.exports = tseslint.config(
  {
    // config with just ignores is the replacement for `.eslintignore`
    ignores: [
      '/dist',
      '/tmp',
      '/out-tsc',
      '/bazel-out',
      'logs',
      '*.log',
      'npm-debug.log*',
      'yarn-debug.log*',
      'yarn-error.log*',
      '/node_modules',
      'npm-debug.log',
      'yarn-error.log',
      '.idea/',
      '.project',
      '.classpath',
      '.c9/',
      '*.launch',
      '.settings/',
      '*.sublime-workspace',
      '.vscode/*',
      '!.vscode/settings.json',
      '!.vscode/tasks.json',
      '!.vscode/launch.json',
      '!.vscode/extensions.json',
      '.history/*',
      '/.angular/cache',
      '.sass-cache/',
      '/connect.lock',
      '/coverage',
      '/libpeerconnection.log',
      'testem.log',
      '/typings',
      '.eslintcache',
      '.cache/',
      '.DS_Store',
      'Thumbs.db',
      '**/src/index.html'
    ]
  },
  {
    files: ['**/*.ts'],
    plugins: { import: importPlugin,deprecate: deprecatePlugin },
    extends: [eslint.configs.recommended, ...tseslint.configs.recommended, ...tseslint.configs.stylistic, ...angular.configs.tsRecommended, prettierConfig],
    processor: angular.processInlineTemplates,
    rules: {
      '@angular-eslint/component-class-suffix': [
        'error',
        {
          suffixes: ['Directive', 'Component', 'Base', 'Widget']
        }
      ],
      '@angular-eslint/directive-class-suffix': [
        'error',
        {
          suffixes: ['Directive', 'Component', 'Base', 'Widget']
        }
      ],
      '@angular-eslint/directive-selector': [
        'error',
        {
          type: 'attribute',
          prefix: ['app', 'cm'],
          style: 'camelCase'
        }
      ],
      '@angular-eslint/component-selector': [
        'error',
        {
          type: 'element',
          prefix: ['app', 'cm'],
          style: 'kebab-case'
        }
      ],
      '@angular-eslint/prefer-standalone': 'off',
      '@angular-eslint/no-attribute-decorator': 'error',
      '@angular-eslint/no-conflicting-lifecycle': 'off',
      '@angular-eslint/no-forward-ref': 'off',
      '@angular-eslint/no-host-metadata-property': 'off',
      '@angular-eslint/no-lifecycle-call': 'off',
      '@angular-eslint/no-pipe-impure': 'error',
      '@angular-eslint/prefer-output-readonly': 'error',
      '@angular-eslint/use-component-selector': 'off',
      '@angular-eslint/use-component-view-encapsulation': 'off',
      '@angular-eslint/no-input-rename': 'off',
      '@angular-eslint/no-output-native': 'off',
      '@angular-eslint/no-empty-lifecycle-method': 'off',
      '@typescript-eslint/array-type': [
        'error',
        {
          default: 'array-simple'
        }
      ],
      '@typescript-eslint/ban-types': [
        'off',
        {
          types: {
            String: {
              message: 'Use string instead.'
            },
            Number: {
              message: 'Use number instead.'
            },
            Boolean: {
              message: 'Use boolean instead.'
            },
            Function: {
              message: 'Use specific callable interface instead.'
            }
          }
        }
      ],
      'import/no-duplicates': 'error',
      'import/no-unused-modules': 'error',
      'import/no-unassigned-import': 'error',
      'import/order': [
        'error',
        {
          alphabetize: { order: 'asc', caseInsensitive: false },
          'newlines-between': 'always',
          groups: ['external', 'internal', ['parent', 'sibling', 'index']],
          pathGroups: [],
          pathGroupsExcludedImportTypes: []
        }
      ],
      '@typescript-eslint/no-unused-vars': 'off',
      '@typescript-eslint/no-this-alias': 'error',
      '@typescript-eslint/member-ordering': 'off',
      '@typescript-eslint/no-empty-function': 'off',
      'no-irregular-whitespace': 'error',
      'no-multiple-empty-lines': 'error',
      'no-sparse-arrays': 'error',
      'prefer-object-spread': 'error',
      'prefer-template': 'error',
      'prefer-const': 'error',
      'max-len': 'off',
      "eqeqeq": ["error", "always"],
      'deprecate/function': 'error',
      'deprecate/member-expression': 'error',
      'deprecate/import': 'error',
    }
  },
  {
    files: ['**/*.html'],
    extends: [...angular.configs.templateRecommended, ...angular.configs.templateAccessibility, prettierConfig],
    rules: {
      '@angular-eslint/template/alt-text': 'off',
      '@angular-eslint/template/click-events-have-key-events': 'off',
      '@angular-eslint/template/interactive-supports-focus': 'off'
    }
  }
);
