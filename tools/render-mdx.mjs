import { readFile, writeFile } from 'node:fs/promises'
import { evaluate } from '@mdx-js/mdx'
import * as runtime from 'react/jsx-runtime'
import { renderToStaticMarkup } from 'react-dom/server'
import React from 'react'

const inputFile = process.argv[2] || 'README.mdx'
const outputFile = process.argv[3] || 'index.html'

try {
  console.log(`Reading ${inputFile}...`)
  const mdx = await readFile(inputFile, 'utf8')
  
  console.log('Compiling MDX...')
  const { default: Content } = await evaluate(mdx, {
    ...runtime,
    baseUrl: import.meta.url,
  })

  console.log('Rendering to HTML...')
  const html = renderToStaticMarkup(React.createElement(Content))

  const doc = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Documentation</title>
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/github-markdown-css/5.5.0/github-markdown.min.css">
  <style>
    .markdown-body {
      box-sizing: border-box;
      min-width: 200px;
      max-width: 980px;
      margin: 0 auto;
      padding: 45px;
    }
    @media (max-width: 767px) {
      .markdown-body {
        padding: 15px;
      }
    }
  </style>
</head>
<body class="markdown-body">
  ${html}
</body>
</html>`

  await writeFile(outputFile, doc)
  console.log(`Successfully compiled ${inputFile} to ${outputFile}`)
} catch (error) {
  console.error('Error compiling MDX:', error)
  process.exit(1)
}
