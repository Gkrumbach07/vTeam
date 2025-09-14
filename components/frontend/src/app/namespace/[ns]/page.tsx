'use client'

import { useState, useEffect } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { SessionList } from '@/components/session-list'
import { NamespaceSelector } from '@/components/ui/namespace-selector'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  UserNamespace,
  UserNamespacesResponse,
} from '@/types/session'
import { getApiClient } from '@/services/api-client'
import {
  Plus,
  Shield,
  AlertTriangle,
  RefreshCw,
  Settings,
  Users,
  DollarSign,
  Activity,
} from 'lucide-react'

export default function NamespacePage() {
  const params = useParams()
  const namespace = params.ns as string

  const [userNamespaces, setUserNamespaces] = useState<UserNamespace[]>([])
  const [currentNamespace, setCurrentNamespace] = useState<UserNamespace | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const apiClient = getApiClient()

  useEffect(() => {
    fetchUserNamespaces()
  }, [])

  useEffect(() => {
    if (userNamespaces.length > 0) {
      const ns = userNamespaces.find(n => n.namespace === namespace)
      setCurrentNamespace(ns || null)

      if (!ns) {
        setError(`Access denied or namespace not found: ${namespace}`)
      }
    }
  }, [namespace, userNamespaces])

  const fetchUserNamespaces = async () => {
    try {
      setLoading(true)
      const response: UserNamespacesResponse = await apiClient.getUserNamespaces()
      setUserNamespaces(response.namespaces || [])
      setError(null)
    } catch (err) {
      console.error('Failed to fetch user namespaces:', err)
      setError('Failed to load namespace information')
      setUserNamespaces([])
    } finally {
      setLoading(false)
    }
  }

  const handleNamespaceChange = (newNamespace: string) => {
    window.location.href = `/namespace/${newNamespace}`
  }

  if (loading) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="flex items-center justify-center py-12">
          <div className="flex items-center space-x-2">
            <RefreshCw className="h-5 w-5 animate-spin" />
            <span>Loading namespace...</span>
          </div>
        </div>
      </div>
    )
  }

  if (error || !currentNamespace) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="text-center py-12">
          <AlertTriangle className="h-8 w-8 text-red-500 mx-auto mb-4" />
          <h1 className="text-2xl font-bold text-gray-900 mb-2">Access Denied</h1>
          <p className="text-red-600 mb-6">
            {error || `You don't have access to the namespace: ${namespace}`}
          </p>
          <div className="space-x-4">
            <Button onClick={fetchUserNamespaces} variant="outline">
              <RefreshCw className="h-4 w-4 mr-2" />
              Retry
            </Button>
            <Link href="/">
              <Button variant="outline">
                Go to Home
              </Button>
            </Link>
          </div>
        </div>
      </div>
    )
  }

  const isEditor = currentNamespace.permission === 'editor'
  const policy = currentNamespace.policy

  return (
    <div className="container mx-auto px-4 py-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">
            {namespace}
          </h1>
          <p className="text-gray-600 mt-1">
            Namespace workspace • {currentNamespace.permission} access
          </p>
        </div>
        <div className="flex items-center space-x-4">
          {/* Namespace Selector */}
          <div className="min-w-64">
            <NamespaceSelector
              onSelect={handleNamespaceChange}
              selectedNamespace={namespace}
            />
          </div>
          {isEditor && (
            <Link href={`/namespace/${namespace}/new`}>
              <Button>
                <Plus className="h-4 w-4 mr-2" />
                New Session
              </Button>
            </Link>
          )}
        </div>
      </div>

      {/* Namespace Info Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center space-x-2">
              <Shield className="h-5 w-5 text-blue-500" />
              <div>
                <p className="text-sm font-medium text-gray-600">Permission</p>
                <div className="flex items-center space-x-2 mt-1">
                  <Badge
                    variant={isEditor ? 'default' : 'secondary'}
                    className={isEditor ? 'bg-green-100 text-green-800' : ''}
                  >
                    {currentNamespace.permission}
                  </Badge>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {policy?.budget && (
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center space-x-2">
                <DollarSign className="h-5 w-5 text-green-500" />
                <div>
                  <p className="text-sm font-medium text-gray-600">Budget</p>
                  <p className="text-lg font-semibold text-gray-900">
                    ${policy.budget}
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {policy?.sessionsActive !== undefined && (
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center space-x-2">
                <Activity className="h-5 w-5 text-blue-500" />
                <div>
                  <p className="text-sm font-medium text-gray-600">Active Sessions</p>
                  <div className="flex items-center space-x-2 mt-1">
                    <span className="text-lg font-semibold text-gray-900">
                      {policy.sessionsActive}
                    </span>
                    {policy.sessionsMax && (
                      <span className="text-sm text-gray-500">
                        / {policy.sessionsMax}
                      </span>
                    )}
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        {policy?.retention && (
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center space-x-2">
                <Settings className="h-5 w-5 text-gray-500" />
                <div>
                  <p className="text-sm font-medium text-gray-600">Retention</p>
                  <p className="text-lg font-semibold text-gray-900">
                    {policy.retention}
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        )}
      </div>

      {/* Sessions List */}
      <SessionList
        namespace={namespace}
        userPermission={currentNamespace.permission}
      />
    </div>
  )
}