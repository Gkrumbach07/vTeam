'use client'

import { useState, useEffect } from 'react'
import { useParams } from 'next/navigation'
import { SessionDetail } from '@/components/session-detail'
import { UserNamespace, UserNamespacesResponse } from '@/types/session'
import { getApiClient } from '@/services/api-client'
import { AlertTriangle, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import Link from 'next/link'

export default function SessionDetailPage() {
  const params = useParams()
  const namespace = params.ns as string
  const sessionName = params.name as string

  const [userPermission, setUserPermission] = useState<'viewer' | 'editor' | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const apiClient = getApiClient()

  useEffect(() => {
    fetchUserPermission()
  }, [namespace])

  const fetchUserPermission = async () => {
    try {
      setLoading(true)
      const response: UserNamespacesResponse = await apiClient.getUserNamespaces()
      const userNamespace = response.namespaces?.find(ns => ns.namespace === namespace)

      if (!userNamespace) {
        throw new Error(`Access denied to namespace: ${namespace}`)
      }

      setUserPermission(userNamespace.permission)
      setError(null)
    } catch (err) {
      console.error('Failed to fetch user permission:', err)
      setError(err instanceof Error ? err.message : 'Failed to verify permissions')
      setUserPermission(null)
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="flex items-center justify-center py-12">
          <div className="flex items-center space-x-2">
            <RefreshCw className="h-5 w-5 animate-spin" />
            <span>Loading session...</span>
          </div>
        </div>
      </div>
    )
  }

  if (error || !userPermission) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="text-center py-12">
          <AlertTriangle className="h-8 w-8 text-red-500 mx-auto mb-4" />
          <h1 className="text-2xl font-bold text-gray-900 mb-2">Access Denied</h1>
          <p className="text-red-600 mb-6">
            {error || 'You don\'t have access to this session'}
          </p>
          <div className="space-x-4">
            <Button onClick={fetchUserPermission} variant="outline">
              <RefreshCw className="h-4 w-4 mr-2" />
              Retry
            </Button>
            <Link href={`/namespace/${namespace}`}>
              <Button variant="outline">
                Back to {namespace}
              </Button>
            </Link>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <SessionDetail
        namespace={namespace}
        sessionName={sessionName}
        userPermission={userPermission}
      />
    </div>
  )
}