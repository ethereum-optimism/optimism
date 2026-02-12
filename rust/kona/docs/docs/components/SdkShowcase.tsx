import React from 'react'

interface SdkProject {
  name: string
  description: string
  linesOfCode: string
  githubUrl: string
}

const projects: SdkProject[] = [
  {
    name: 'Kona Client',
    description: 'Fault proof program for rollup state transitions',
    linesOfCode: '~3K LoC',
    githubUrl: 'https://github.com/op-rs/kona/tree/main/bin/client'
  },
  {
    name: 'Kona Node',
    description: 'Modular OP Stack rollup node implementation',
    linesOfCode: '~8K LoC',
    githubUrl: 'https://github.com/op-rs/kona/tree/main/bin/node'
  },
  {
    name: 'OP Succinct',
    description: 'zkVM-based proof system using Kona',
    linesOfCode: '~2K LoC',
    githubUrl: 'https://github.com/succinctlabs/op-succinct'
  },
  {
    name: 'Kailua',
    description: 'zkVM-based proof system using Kona',
    linesOfCode: '~5K LoC',
    githubUrl: 'https://github.com/boundless-xyz/kailua'
  }
]

export function SdkShowcase() {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
      {projects.map((project, index) => (
        <div
          key={index}
          className="relative bg-white/5 dark:bg-white/5 border border-black/10 dark:border-white/10 rounded-xl p-6 hover:bg-black/5 dark:hover:bg-white/10 transition-colors group"
        >
          <div className="flex items-start justify-between mb-4">
            <h3 className="text-lg font-semibold text-black dark:text-white">{project.name}</h3>
            <a
              href={project.githubUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="text-gray-500 hover:text-black dark:hover:text-white transition-colors"
            >
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
              </svg>
            </a>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm mb-4">{project.description}</p>
          <div className="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200">
            {project.linesOfCode}
          </div>
        </div>
      ))}
    </div>
  )
}
