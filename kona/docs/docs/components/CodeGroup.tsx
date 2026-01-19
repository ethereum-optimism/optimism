import React, { useState } from 'react'

interface CodeProps {
  title: string
  code: string
}

interface CodeGroupProps {
  children: React.ReactElement<CodeProps>[]
}

export function CodeGroup({ children }: CodeGroupProps) {
  const [activeTab, setActiveTab] = useState(0)

  return (
    <div className="w-full max-w-3xl">
      <div className="flex border-b border-gray-200 dark:border-gray-700 mb-6">
        {children.map((child, index) => (
          <button
            key={index}
            onClick={() => setActiveTab(index)}
            className={`px-6 py-4 text-base font-medium transition-colors ${
              activeTab === index
                ? 'border-b-2 border-blue-500 text-blue-600 dark:text-blue-400'
                : 'text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200'
            }`}
          >
            {child.props.title}
          </button>
        ))}
      </div>
      <div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-8 border border-gray-200 dark:border-gray-700 shadow-sm">
        <pre className="text-base overflow-x-auto">
          <code className="text-gray-800 dark:text-gray-200 leading-relaxed">
            {children[activeTab].props.code}
          </code>
        </pre>
      </div>
    </div>
  )
}

export function Code({ title, code }: CodeProps) {
  return null // This component is only used as a child of CodeGroup
}