'use client'

import { useState, useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { SessionTypeSelector } from '@/components/ui/session-type-selector'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Label } from '@/components/ui/label'
import {
  CreateSessionRequest,
  UserNamespace,
  UserNamespacesResponse,
  FrameworkType,
} from '@/types/session'
import { getApiClient } from '@/services/api-client'
import {
  ArrowLeft,
  Save,
  AlertTriangle,
  RefreshCw,
  Wand2,
  Settings,
  Shield,
  Cpu,
  DollarSign,
  Wrench,
} from 'lucide-react'

export default function NewSessionPage() {
  const params = useParams()
  const router = useRouter()
  const namespace = params.ns as string

  const [userNamespace, setUserNamespace] = useState<UserNamespace | null>(null)
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Form state
  const [formData, setFormData] = useState<{
    frameworkType: FrameworkType | ''
    frameworkVersion: string
    config: Record<string, any>
    triggerSource: string
    triggerEvent: string
    model?: string
    budget?: string
    approvalRequired: boolean
  }>({
    frameworkType: '',
    frameworkVersion: 'latest',
    config: {},
    triggerSource: 'manual',
    triggerEvent: 'create',
    approvalRequired: false,
  })

  const apiClient = getApiClient()

  useEffect(() => {
    fetchUserPermission()
  }, [namespace])

  const fetchUserPermission = async () => {
    try {
      setLoading(true)
      const response: UserNamespacesResponse = await apiClient.getUserNamespaces()
      const ns = response.namespaces?.find(n => n.namespace === namespace)

      if (!ns) {
        throw new Error(`Access denied to namespace: ${namespace}`)
      }

      if (ns.permission !== 'editor') {
        throw new Error('You need editor permission to create sessions')
      }

      setUserNamespace(ns)
      setError(null)

      // Set default budget if available
      if (ns.policy?.budget) {
        setFormData(prev => ({ ...prev, budget: ns.policy?.budget }))
      }
    } catch (err) {
      console.error('Failed to fetch user permission:', err)
      setError(err instanceof Error ? err.message : 'Failed to verify permissions')
      setUserNamespace(null)
    } finally {
      setLoading(false)
    }
  }

  const handleFrameworkTypeChange = (type: string) => {
    const frameworkType = type as FrameworkType
    setFormData(prev => ({
      ...prev,
      frameworkType,
      config: frameworkType === 'claude-code' ? { model: 'claude-3-sonnet' } : {},
    }))
  }

  const handleConfigChange = (key: string, value: any) => {
    setFormData(prev => ({
      ...prev,
      config: {
        ...prev.config,
        [key]: value,
      },
    }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!formData.frameworkType) {
      setError('Please select a framework type')
      return
    }

    try {
      setCreating(true)
      setError(null)

      const sessionData: CreateSessionRequest = {
        trigger: {
          source: formData.triggerSource as any,
          event: formData.triggerEvent,
          payload: {},
        },
        framework: {
          type: formData.frameworkType,
          version: formData.frameworkVersion,
          config: formData.config,
        },
        policy: {
          modelConstraints: {
            allowed: formData.config.model ? [formData.config.model] : ['claude-3-sonnet'],
            budget: formData.budget || userNamespace?.policy?.budget || '100.00',
          },
          toolConstraints: {
            allowed: ['bash', 'read', 'write', 'edit', 'grep', 'glob'],
          },
          approvalRequired: formData.approvalRequired,
        },
      }

      const session = await apiClient.createSession(namespace, sessionData)

      // Redirect to the new session
      router.push(`/namespace/${namespace}/session/${session.metadata.name}`)
    } catch (err) {
      console.error('Failed to create session:', err)
      setError(err instanceof Error ? err.message : 'Failed to create session')
    } finally {
      setCreating(false)
    }
  }

  if (loading) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="flex items-center justify-center py-12">
          <div className="flex items-center space-x-2">
            <RefreshCw className="h-5 w-5 animate-spin" />
            <span>Loading...</span>
          </div>
        </div>
      </div>
    )
  }

  if (error || !userNamespace) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="text-center py-12">
          <AlertTriangle className="h-8 w-8 text-red-500 mx-auto mb-4" />
          <h1 className="text-2xl font-bold text-gray-900 mb-2">Access Denied</h1>
          <p className="text-red-600 mb-6">
            {error || 'You don\'t have permission to create sessions'}
          </p>
          <div className="space-x-4">
            <Button onClick={fetchUserPermission} variant="outline">
              <RefreshCw className="h-4 w-4 mr-2" />
              Retry
            </Button>
            <Link href={`/namespace/${namespace}`}>
              <Button variant="outline">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to {namespace}
              </Button>
            </Link>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto px-4 py-8 max-w-4xl">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div className="flex items-center space-x-4">
          <Link href={`/namespace/${namespace}`}>
            <Button variant="outline" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back
            </Button>
          </Link>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Create New Session</h1>
            <p className="text-gray-600">
              Create a new agentic session in {namespace}
            </p>
          </div>
        </div>
      </div>

      {/* Namespace Info */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Shield className="h-5 w-5" />
            <span>Namespace: {namespace}</span>
          </CardTitle>
          <CardDescription>
            Current policy constraints and permissions
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="flex items-center space-x-2">
              <DollarSign className="h-4 w-4 text-green-500" />
              <div>
                <p className="text-sm font-medium text-gray-600">Budget</p>
                <p className="text-sm text-gray-900">
                  ${userNamespace.policy?.budget || '100.00'}
                </p>
              </div>
            </div>
            <div className="flex items-center space-x-2">
              <Cpu className="h-4 w-4 text-blue-500" />
              <div>
                <p className="text-sm font-medium text-gray-600">Active Sessions</p>
                <p className="text-sm text-gray-900">
                  {userNamespace.policy?.sessionsActive || 0}
                  {userNamespace.policy?.sessionsMax && ` / ${userNamespace.policy.sessionsMax}`}
                </p>
              </div>
            </div>
            <div className="flex items-center space-x-2">
              <Settings className="h-4 w-4 text-gray-500" />
              <div>
                <p className="text-sm font-medium text-gray-600">Permission</p>
                <Badge className="bg-green-100 text-green-800">
                  {userNamespace.permission}
                </Badge>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Create Session Form */}
      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Framework Selection */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <Wand2 className="h-5 w-5" />
              <span>Framework Type</span>
            </CardTitle>
            <CardDescription>
              Choose the agentic framework for this session
            </CardDescription>
          </CardHeader>
          <CardContent>
            <SessionTypeSelector
              onSelect={handleFrameworkTypeChange}
              selectedType={formData.frameworkType}
            />
          </CardContent>
        </Card>

        {/* Framework Configuration */}
        {formData.frameworkType && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center space-x-2">
                <Settings className="h-5 w-5" />
                <span>Configuration</span>
              </CardTitle>
              <CardDescription>
                Configure the framework settings
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="version">Framework Version</Label>
                  <select
                    id="version"
                    value={formData.frameworkVersion}
                    onChange={(e) => setFormData(prev => ({ ...prev, frameworkVersion: e.target.value }))}
                    className="w-full mt-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="latest">Latest</option>
                    <option value="v1.0">v1.0</option>
                    <option value="v1.1">v1.1</option>
                  </select>
                </div>

                {formData.frameworkType === 'claude-code' && (
                  <div>
                    <Label htmlFor="model">Model</Label>
                    <select
                      id="model"
                      value={formData.config.model || 'claude-3-sonnet'}
                      onChange={(e) => handleConfigChange('model', e.target.value)}
                      className="w-full mt-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                      <option value="claude-3-sonnet">Claude 3 Sonnet</option>
                      <option value="claude-3-haiku">Claude 3 Haiku</option>
                      <option value="claude-3-opus">Claude 3 Opus</option>
                    </select>
                  </div>
                )}
              </div>

              <div>
                <Label htmlFor="budget">Budget Override (optional)</Label>
                <input
                  id="budget"
                  type="number"
                  step="0.01"
                  min="0"
                  placeholder={userNamespace.policy?.budget || '100.00'}
                  value={formData.budget || ''}
                  onChange={(e) => setFormData(prev => ({ ...prev, budget: e.target.value }))}
                  className="w-full mt-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-gray-500 mt-1">
                  Leave empty to use namespace default: ${userNamespace.policy?.budget || '100.00'}
                </p>
              </div>

              <div className="flex items-center space-x-2">
                <input
                  id="approval"
                  type="checkbox"
                  checked={formData.approvalRequired}
                  onChange={(e) => setFormData(prev => ({ ...prev, approvalRequired: e.target.checked }))}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <Label htmlFor="approval" className="text-sm">
                  Require approval before execution
                </Label>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Actions */}
        <div className="flex items-center justify-between">
          <Link href={`/namespace/${namespace}`}>
            <Button type="button" variant="outline">
              Cancel
            </Button>
          </Link>
          <Button
            type="submit"
            disabled={creating || !formData.frameworkType}
          >
            {creating ? (
              <>
                <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                Creating...
              </>
            ) : (
              <>
                <Save className="h-4 w-4 mr-2" />
                Create Session
              </>
            )}
          </Button>
        </div>

        {error && (
          <div className="flex items-center space-x-2 p-3 border border-red-200 rounded-lg bg-red-50">
            <AlertTriangle className="h-4 w-4 text-red-500" />
            <span className="text-sm text-red-600">{error}</span>
          </div>
        )}
      </form>
    </div>
  )
}