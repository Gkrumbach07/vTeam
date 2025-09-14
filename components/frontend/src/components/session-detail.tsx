'use client'

import { useState, useEffect, useCallback } from 'react'
import { formatDistanceToNow, format } from 'date-fns'
import Link from 'next/link'
import {
  Session,
  SessionCondition,
  HistoryEvent,
  getSessionDisplayName,
  getSessionDescription,
  formatDuration,
  getPhaseColor
} from '@/types/session'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs'
import {
  ArrowLeft,
  RefreshCw,
  Clock,
  Play,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Square,
  Trash2,
  ExternalLink,
  Settings,
  History,
  FileText,
  Activity,
  Shield,
  Cpu,
  Memory,
} from 'lucide-react'

interface SessionDetailProps {
  namespace: string
  sessionName: string
  userPermission: 'viewer' | 'editor'
}

interface SessionDetailState {
  session: Session | null
  loading: boolean
  error: string | null
  logs: string[]
  artifacts: Array<{
    name: string
    type: string
    size: string
    url: string
  }>
}

const CONDITION_ICONS = {
  PolicyValidated: Shield,
  WorkloadCreated: Settings,
  WorkloadRunning: Activity,
  ArtifactsStored: FileText,
}

const PHASE_ICONS = {
  Pending: Clock,
  Running: Play,
  Completed: CheckCircle,
  Failed: XCircle,
}

export function SessionDetail({ namespace, sessionName, userPermission }: SessionDetailProps) {
  const [state, setState] = useState<SessionDetailState>({
    session: null,
    loading: true,
    error: null,
    logs: [],
    artifacts: [],
  })
  const [actionLoading, setActionLoading] = useState<string | null>(null)

  const fetchSession = useCallback(async () => {
    try {
      setState(prev => ({ ...prev, loading: true, error: null }))

      const token = localStorage.getItem('authToken')
      const response = await fetch(`/api/v1/namespaces/${namespace}/sessions/${sessionName}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      })

      if (!response.ok) {
        if (response.status === 404) {
          throw new Error('Session not found')
        }
        if (response.status === 403) {
          throw new Error(`Access denied to session in namespace: ${namespace}`)
        }
        throw new Error(`Failed to fetch session: ${response.statusText}`)
      }

      const session: Session = await response.json()

      setState(prev => ({
        ...prev,
        session,
        loading: false,
        error: null,
      }))
    } catch (err) {
      setState(prev => ({
        ...prev,
        loading: false,
        error: err instanceof Error ? err.message : 'Unknown error',
      }))
    }
  }, [namespace, sessionName])

  const fetchLogs = useCallback(async () => {
    if (!state.session) return

    try {
      const token = localStorage.getItem('authToken')
      const response = await fetch(`/api/v1/namespaces/${namespace}/sessions/${sessionName}/logs`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      })

      if (response.ok) {
        const data = await response.json()
        setState(prev => ({ ...prev, logs: data.logs || [] }))
      }
    } catch (err) {
      console.error('Failed to fetch logs:', err)
    }
  }, [namespace, sessionName, state.session])

  const fetchArtifacts = useCallback(async () => {
    if (!state.session) return

    try {
      const token = localStorage.getItem('authToken')
      const response = await fetch(`/api/v1/namespaces/${namespace}/sessions/${sessionName}/artifacts`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      })

      if (response.ok) {
        const data = await response.json()
        setState(prev => ({ ...prev, artifacts: data.artifacts || [] }))
      }
    } catch (err) {
      console.error('Failed to fetch artifacts:', err)
    }
  }, [namespace, sessionName, state.session])

  useEffect(() => {
    fetchSession()

    // Poll for updates every 10 seconds
    const interval = setInterval(fetchSession, 10000)
    return () => clearInterval(interval)
  }, [fetchSession])

  useEffect(() => {
    if (state.session) {
      fetchLogs()
      fetchArtifacts()
    }
  }, [state.session, fetchLogs, fetchArtifacts])

  const handleSessionAction = async (action: 'stop' | 'delete') => {
    if (!state.session || userPermission === 'viewer') return

    try {
      setActionLoading(action)

      const token = localStorage.getItem('authToken')
      const endpoint = action === 'stop'
        ? `/api/v1/namespaces/${namespace}/sessions/${sessionName}/stop`
        : `/api/v1/namespaces/${namespace}/sessions/${sessionName}`

      const response = await fetch(endpoint, {
        method: action === 'delete' ? 'DELETE' : 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      })

      if (!response.ok) {
        throw new Error(`Failed to ${action} session: ${response.statusText}`)
      }

      if (action === 'delete') {
        // Redirect to session list
        window.location.href = `/namespace/${namespace}`
      } else {
        // Refresh session data
        fetchSession()
      }
    } catch (err) {
      console.error(`Failed to ${action} session:`, err)
      // TODO: Show toast notification
    } finally {
      setActionLoading(null)
    }
  }

  const getConditionIcon = (type: string) => {
    const Icon = CONDITION_ICONS[type as keyof typeof CONDITION_ICONS]
    return Icon ? <Icon className="h-4 w-4" /> : <Activity className="h-4 w-4" />
  }

  const getPhaseIcon = (phase: string) => {
    const Icon = PHASE_ICONS[phase as keyof typeof PHASE_ICONS]
    return Icon ? <Icon className="h-4 w-4" /> : <Clock className="h-4 w-4" />
  }

  const formatConditionStatus = (condition: SessionCondition) => {
    const isTrue = condition.status === 'True'
    return (
      <div className={`flex items-center space-x-2 ${isTrue ? 'text-green-600' : 'text-red-600'}`}>
        {getConditionIcon(condition.type)}
        <span className="font-medium">{condition.type}</span>
        <Badge
          variant={isTrue ? 'default' : 'destructive'}
          className="text-xs"
        >
          {condition.status}
        </Badge>
      </div>
    )
  }

  if (state.loading && !state.session) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="flex items-center space-x-2">
          <RefreshCw className="h-5 w-5 animate-spin" />
          <span>Loading session...</span>
        </div>
      </div>
    )
  }

  if (state.error) {
    return (
      <div className="text-center py-12">
        <AlertTriangle className="h-8 w-8 text-red-500 mx-auto mb-2" />
        <p className="text-red-600 mb-4">{state.error}</p>
        <div className="space-x-2">
          <Button onClick={fetchSession} variant="outline" size="sm">
            <RefreshCw className="h-4 w-4 mr-2" />
            Retry
          </Button>
          <Link href={`/namespace/${namespace}`}>
            <Button variant="outline" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Sessions
            </Button>
          </Link>
        </div>
      </div>
    )
  }

  if (!state.session) {
    return null
  }

  const session = state.session
  const phase = session.status?.phase || 'Pending'
  const isRunning = phase === 'Running'
  const canStop = isRunning && userPermission === 'editor'
  const canDelete = userPermission === 'editor'

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Link href={`/namespace/${namespace}`}>
            <Button variant="outline" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back
            </Button>
          </Link>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">
              {getSessionDisplayName(session)}
            </h1>
            <p className="text-sm text-gray-500">
              {getSessionDescription(session)} • {namespace}
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <Button onClick={fetchSession} variant="outline" size="sm" disabled={state.loading}>
            <RefreshCw className={`h-4 w-4 mr-2 ${state.loading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
          {canStop && (
            <Button
              onClick={() => handleSessionAction('stop')}
              disabled={actionLoading === 'stop'}
              variant="outline"
              size="sm"
              className="text-orange-600 hover:text-orange-700"
            >
              <Square className="h-4 w-4 mr-2" />
              Stop
            </Button>
          )}
          {canDelete && (
            <Button
              onClick={() => handleSessionAction('delete')}
              disabled={actionLoading === 'delete'}
              variant="outline"
              size="sm"
              className="text-red-600 hover:text-red-700"
            >
              <Trash2 className="h-4 w-4 mr-2" />
              Delete
            </Button>
          )}
        </div>
      </div>

      {/* Status Overview */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center space-x-2">
              {getPhaseIcon(phase)}
              <div>
                <p className="text-sm font-medium text-gray-600">Status</p>
                <Badge className={getPhaseColor(phase)}>
                  {phase}
                </Badge>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center space-x-2">
              <Cpu className="h-4 w-4 text-gray-500" />
              <div>
                <p className="text-sm font-medium text-gray-600">Framework</p>
                <p className="text-sm text-gray-900">
                  {session.spec.framework.type} v{session.spec.framework.version}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center space-x-2">
              <Clock className="h-4 w-4 text-gray-500" />
              <div>
                <p className="text-sm font-medium text-gray-600">Duration</p>
                <p className="text-sm text-gray-900">
                  {formatDuration(
                    session.status?.startTime,
                    session.status?.completionTime
                  )}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center space-x-2">
              <History className="h-4 w-4 text-gray-500" />
              <div>
                <p className="text-sm font-medium text-gray-600">Created</p>
                <p className="text-sm text-gray-900">
                  {formatDistanceToNow(new Date(session.metadata.creationTimestamp), { addSuffix: true })}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Main Content */}
      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="conditions">Conditions</TabsTrigger>
          <TabsTrigger value="history">History</TabsTrigger>
          <TabsTrigger value="logs">Logs</TabsTrigger>
          <TabsTrigger value="artifacts">Artifacts</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Trigger Information */}
            <Card>
              <CardHeader>
                <CardTitle>Trigger</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div>
                  <span className="text-sm font-medium text-gray-600">Source:</span>
                  <span className="ml-2 text-sm text-gray-900">{session.spec.trigger.source}</span>
                </div>
                <div>
                  <span className="text-sm font-medium text-gray-600">Event:</span>
                  <span className="ml-2 text-sm text-gray-900">{session.spec.trigger.event}</span>
                </div>
                {session.spec.trigger.payload && Object.keys(session.spec.trigger.payload).length > 0 && (
                  <div>
                    <span className="text-sm font-medium text-gray-600">Payload:</span>
                    <pre className="mt-1 text-xs bg-gray-50 p-2 rounded overflow-x-auto">
                      {JSON.stringify(session.spec.trigger.payload, null, 2)}
                    </pre>
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Policy Information */}
            <Card>
              <CardHeader>
                <CardTitle>Policy</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div>
                  <span className="text-sm font-medium text-gray-600">Allowed Models:</span>
                  <div className="mt-1 flex flex-wrap gap-1">
                    {session.spec.policy.modelConstraints.allowed.map((model, index) => (
                      <Badge key={index} variant="secondary" className="text-xs">
                        {model}
                      </Badge>
                    ))}
                  </div>
                </div>
                <div>
                  <span className="text-sm font-medium text-gray-600">Budget:</span>
                  <span className="ml-2 text-sm text-gray-900">${session.spec.policy.modelConstraints.budget}</span>
                </div>
                <div>
                  <span className="text-sm font-medium text-gray-600">Allowed Tools:</span>
                  <div className="mt-1 flex flex-wrap gap-1">
                    {session.spec.policy.toolConstraints.allowed.map((tool, index) => (
                      <Badge key={index} variant="outline" className="text-xs">
                        {tool}
                      </Badge>
                    ))}
                  </div>
                </div>
                <div>
                  <span className="text-sm font-medium text-gray-600">Approval Required:</span>
                  <span className="ml-2 text-sm text-gray-900">
                    {session.spec.policy.approvalRequired ? 'Yes' : 'No'}
                  </span>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="conditions" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Session Conditions</CardTitle>
              <CardDescription>
                Current status of session validation and execution conditions
              </CardDescription>
            </CardHeader>
            <CardContent>
              {session.status?.conditions && session.status.conditions.length > 0 ? (
                <div className="space-y-4">
                  {session.status.conditions.map((condition, index) => (
                    <div key={index} className="flex items-center justify-between p-3 border rounded-lg">
                      <div className="flex items-center space-x-3">
                        {formatConditionStatus(condition)}
                      </div>
                      <div className="text-right">
                        <div className="text-xs text-gray-500">
                          {format(new Date(condition.lastTransitionTime), 'MMM d, HH:mm:ss')}
                        </div>
                        {condition.message && (
                          <div className="text-xs text-gray-600 mt-1">
                            {condition.message}
                          </div>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-gray-500">No conditions reported</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="history" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Session History</CardTitle>
              <CardDescription>
                Chronological history of session events
              </CardDescription>
            </CardHeader>
            <CardContent>
              {session.status?.history && session.status.history.length > 0 ? (
                <div className="space-y-3">
                  {session.status.history.map((event, index) => (
                    <div key={index} className="flex items-start space-x-3 p-3 border rounded-lg">
                      <Activity className="h-4 w-4 text-blue-500 mt-0.5 flex-shrink-0" />
                      <div className="flex-1">
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-medium text-gray-900">{event.event}</span>
                          <span className="text-xs text-gray-500">
                            {format(new Date(event.timestamp), 'MMM d, HH:mm:ss')}
                          </span>
                        </div>
                        {event.data && Object.keys(event.data).length > 0 && (
                          <pre className="mt-1 text-xs bg-gray-50 p-2 rounded overflow-x-auto">
                            {JSON.stringify(event.data, null, 2)}
                          </pre>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-gray-500">No history events</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="logs" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Session Logs</CardTitle>
              <CardDescription>
                Real-time logs from the session workload
              </CardDescription>
            </CardHeader>
            <CardContent>
              {state.logs.length > 0 ? (
                <pre className="text-xs bg-gray-900 text-green-400 p-4 rounded-lg overflow-x-auto max-h-96">
                  {state.logs.join('\n')}
                </pre>
              ) : (
                <p className="text-sm text-gray-500">No logs available</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="artifacts" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Artifacts</CardTitle>
              <CardDescription>
                Generated artifacts and outputs from the session
              </CardDescription>
            </CardHeader>
            <CardContent>
              {state.artifacts.length > 0 ? (
                <div className="space-y-2">
                  {state.artifacts.map((artifact, index) => (
                    <div key={index} className="flex items-center justify-between p-3 border rounded-lg">
                      <div className="flex items-center space-x-3">
                        <FileText className="h-4 w-4 text-gray-500" />
                        <div>
                          <div className="text-sm font-medium text-gray-900">{artifact.name}</div>
                          <div className="text-xs text-gray-500">{artifact.type} • {artifact.size}</div>
                        </div>
                      </div>
                      <Button variant="outline" size="sm" asChild>
                        <a href={artifact.url} target="_blank" rel="noopener noreferrer">
                          <ExternalLink className="h-4 w-4 mr-2" />
                          View
                        </a>
                      </Button>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-gray-500">No artifacts generated</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}