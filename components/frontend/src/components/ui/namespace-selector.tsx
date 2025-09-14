'use client'

import { useState, useEffect } from 'react'
import { ChevronDown, Search, Shield, Edit } from 'lucide-react'

interface Namespace {
  namespace: string
  permission: 'viewer' | 'editor'
  policy?: {
    budget?: string
    sessionsActive?: number
  }
}

interface NamespaceSelectorProps {
  onSelect: (namespace: string) => void
  selectedNamespace?: string
}

export function NamespaceSelector({ onSelect, selectedNamespace }: NamespaceSelectorProps) {
  const [namespaces, setNamespaces] = useState<Namespace[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [isOpen, setIsOpen] = useState(false)

  useEffect(() => {
    fetchNamespaces()
  }, [])

  const fetchNamespaces = async () => {
    try {
      setLoading(true)
      const token = localStorage.getItem('authToken') // OAuth token
      const response = await fetch('/api/v1/user/namespaces', {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      })

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      const data = await response.json()
      setNamespaces(data.namespaces || [])
      setError(null)
    } catch (err) {
      console.error('Failed to fetch namespaces:', err)
      setError('Error loading namespaces')
      setNamespaces([])
    } finally {
      setLoading(false)
    }
  }

  const filteredNamespaces = namespaces.filter(ns =>
    ns.namespace.toLowerCase().includes(searchTerm.toLowerCase())
  )

  const handleSelect = (namespace: string) => {
    onSelect(namespace)
    setIsOpen(false)
  }

  if (loading) {
    return (
      <div className="flex items-center space-x-2 p-3 border rounded-lg bg-gray-50">
        <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-gray-600"></div>
        <span className="text-sm text-gray-600">Loading namespaces...</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center space-x-2 p-3 border rounded-lg bg-red-50 border-red-200">
        <Shield className="h-4 w-4 text-red-500" />
        <span className="text-sm text-red-600">{error}</span>
        <button
          onClick={fetchNamespaces}
          className="ml-auto text-xs text-red-500 hover:text-red-700 underline"
        >
          Retry
        </button>
      </div>
    )
  }

  if (namespaces.length === 0) {
    return (
      <div className="flex items-center space-x-2 p-3 border rounded-lg bg-yellow-50 border-yellow-200">
        <Shield className="h-4 w-4 text-yellow-500" />
        <span className="text-sm text-yellow-700">No namespaces available</span>
      </div>
    )
  }

  const selectedNs = namespaces.find(ns => ns.namespace === selectedNamespace)

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center justify-between w-full p-3 border rounded-lg bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
      >
        <div className="flex items-center space-x-3">
          <Shield className="h-4 w-4 text-gray-500" />
          <div className="text-left">
            <div className="text-sm font-medium text-gray-900">
              {selectedNs?.namespace || 'Select Namespace'}
            </div>
            {selectedNs && (
              <div className="flex items-center space-x-2 text-xs text-gray-500">
                <Edit className="h-3 w-3" />
                <span className="capitalize">{selectedNs.permission}</span>
                {selectedNs.policy?.sessionsActive !== undefined && (
                  <span>• {selectedNs.policy.sessionsActive} active sessions</span>
                )}
              </div>
            )}
          </div>
        </div>
        <ChevronDown
          className={`h-4 w-4 text-gray-400 transition-transform ${
            isOpen ? 'transform rotate-180' : ''
          }`}
        />
      </button>

      {isOpen && (
        <div className="absolute z-50 w-full mt-1 bg-white border rounded-lg shadow-lg max-h-64 overflow-hidden">
          <div className="p-2 border-b">
            <div className="relative">
              <Search className="absolute left-2 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-400" />
              <input
                type="text"
                placeholder="Search namespaces..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-8 pr-3 py-2 text-sm border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>

          <div className="max-h-48 overflow-y-auto">
            {filteredNamespaces.length === 0 ? (
              <div className="p-3 text-sm text-gray-500 text-center">
                No namespaces match your search
              </div>
            ) : (
              filteredNamespaces.map((ns) => (
                <button
                  key={ns.namespace}
                  onClick={() => handleSelect(ns.namespace)}
                  className={`w-full text-left p-3 hover:bg-gray-50 border-b border-gray-100 last:border-b-0 ${
                    selectedNamespace === ns.namespace ? 'bg-blue-50 border-blue-200' : ''
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="text-sm font-medium text-gray-900">
                        {ns.namespace}
                      </div>
                      <div className="flex items-center space-x-2 text-xs text-gray-500 mt-1">
                        {ns.permission === 'editor' ? (
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs bg-green-100 text-green-800">
                            <Edit className="h-3 w-3 mr-1" />
                            Editor
                          </span>
                        ) : (
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs bg-gray-100 text-gray-600">
                            <Shield className="h-3 w-3 mr-1" />
                            Viewer
                          </span>
                        )}
                        {ns.policy?.budget && (
                          <span className="text-xs text-gray-500">
                            Budget: ${ns.policy.budget}
                          </span>
                        )}
                      </div>
                    </div>
                    {ns.policy?.sessionsActive !== undefined && (
                      <div className="text-right">
                        <div className="text-xs text-gray-500">Active Sessions</div>
                        <div className="text-sm font-medium text-gray-900">
                          {ns.policy.sessionsActive}
                        </div>
                      </div>
                    )}
                  </div>
                </button>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  )
}