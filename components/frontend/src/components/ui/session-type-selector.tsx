'use client'

import { useState } from 'react'
import { ChevronDown, Code, Wrench, Cpu } from 'lucide-react'

interface SessionType {
  type: string
  name: string
  description: string
  icon: React.ReactNode
  capabilities: string[]
  resources: {
    cpu: string
    memory: string
  }
}

interface SessionTypeSelectorProps {
  onSelect: (sessionType: string) => void
  selectedType?: string
}

const sessionTypes: SessionType[] = [
  {
    type: 'claude-code',
    name: 'Claude Code',
    description: 'AI-powered code analysis and automation with browser capabilities',
    icon: <Code className="h-4 w-4" />,
    capabilities: [
      'Browser automation with Playwright',
      'Website analysis and screenshots',
      'Code generation and editing',
      'File system operations',
      'Multi-step task execution'
    ],
    resources: {
      cpu: '1000m',
      memory: '2Gi'
    }
  },
  {
    type: 'generic',
    name: 'Generic Runner',
    description: 'General-purpose framework-agnostic runner for custom tasks',
    icon: <Wrench className="h-4 w-4" />,
    capabilities: [
      'Custom framework support',
      'Configurable execution environment',
      'Basic file operations',
      'Environment variable support'
    ],
    resources: {
      cpu: '500m',
      memory: '1Gi'
    }
  }
]

export function SessionTypeSelector({ onSelect, selectedType }: SessionTypeSelectorProps) {
  const [isOpen, setIsOpen] = useState(false)

  const handleSelect = (type: string) => {
    onSelect(type)
    setIsOpen(false)
  }

  const selectedSessionType = sessionTypes.find(st => st.type === selectedType)

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center justify-between w-full p-3 border rounded-lg bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
      >
        <div className="flex items-center space-x-3">
          <div className="flex-shrink-0">
            {selectedSessionType?.icon || <Cpu className="h-4 w-4 text-gray-500" />}
          </div>
          <div className="text-left">
            <div className="text-sm font-medium text-gray-900">
              {selectedSessionType?.name || 'Select Session Type'}
            </div>
            {selectedSessionType && (
              <div className="text-xs text-gray-500 mt-1">
                {selectedSessionType.description}
              </div>
            )}
          </div>
        </div>
        <ChevronDown
          className={`h-4 w-4 text-gray-400 transition-transform flex-shrink-0 ${
            isOpen ? 'transform rotate-180' : ''
          }`}
        />
      </button>

      {isOpen && (
        <div className="absolute z-50 w-full mt-1 bg-white border rounded-lg shadow-lg max-h-96 overflow-hidden">
          <div className="max-h-80 overflow-y-auto">
            {sessionTypes.map((sessionType) => (
              <button
                key={sessionType.type}
                onClick={() => handleSelect(sessionType.type)}
                className={`w-full text-left p-4 hover:bg-gray-50 border-b border-gray-100 last:border-b-0 transition-colors ${
                  selectedType === sessionType.type ? 'bg-blue-50 border-blue-200' : ''
                }`}
              >
                <div className="flex items-start space-x-3">
                  <div className="flex-shrink-0 mt-0.5">
                    {sessionType.icon}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between">
                      <div className="text-sm font-medium text-gray-900">
                        {sessionType.name}
                      </div>
                      <div className="flex items-center space-x-2 text-xs text-gray-500">
                        <Cpu className="h-3 w-3" />
                        <span>{sessionType.resources.cpu} / {sessionType.resources.memory}</span>
                      </div>
                    </div>
                    <div className="text-xs text-gray-600 mt-1 line-clamp-2">
                      {sessionType.description}
                    </div>
                    <div className="mt-2">
                      <div className="text-xs font-medium text-gray-700 mb-1">Capabilities:</div>
                      <div className="flex flex-wrap gap-1">
                        {sessionType.capabilities.slice(0, 3).map((capability, index) => (
                          <span
                            key={index}
                            className="inline-flex items-center px-2 py-0.5 rounded text-xs bg-gray-100 text-gray-600"
                          >
                            {capability}
                          </span>
                        ))}
                        {sessionType.capabilities.length > 3 && (
                          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs bg-gray-100 text-gray-600">
                            +{sessionType.capabilities.length - 3} more
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

// Export session type information for use in other components
export { sessionTypes }
export type { SessionType }