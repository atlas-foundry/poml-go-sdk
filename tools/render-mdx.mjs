import { readFile, writeFile } from 'node:fs/promises'
import { marked } from 'marked'

const inputFile = process.argv[2] || 'README.md'
const outputFile = process.argv[3] || 'index.html'

try {
  console.log(`Reading ${inputFile}...`)
  const markdown = await readFile(inputFile, 'utf8')

  console.log('Rendering Markdown to HTML...')
  const html = marked(markdown)

  const doc = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>POML Go SDK Documentation</title>
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
  console.log(`Successfully rendered ${inputFile} to ${outputFile}`)
} catch (error) {
  console.error('Error rendering Markdown:', error)
  process.exit(1)
}
