'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { formatDistanceToNow } from 'date-fns'
import {
  Session,
  SessionPhase,
  UserNamespace,
  SessionListResponse,
  getSessionDisplayName,
  getSessionDescription,
  formatDuration,
  getPhaseColor
} from '@/types/session'
import { Badge } from '@/components/ui/badge'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import {
  RefreshCw,
  MoreHorizontal,
  Square,
  Trash2,
  Eye,
  Play,
  AlertTriangle,
  Clock,
  CheckCircle,
  XCircle,
  Filter,
  Search,
} from 'lucide-react'

interface SessionListProps {
  namespace: string
  userPermission: 'viewer' | 'editor'
}

interface SessionListState {
  sessions: Session[]
  loading: boolean
  error: string | null
  hasMore: boolean
  nextPageToken?: string
}

interface FilterOptions {
  phase: SessionPhase | 'all'
  framework: string | 'all'
  searchTerm: string
}

const PHASE_ICONS = {
  Pending: Clock,
  Running: Play,
  Completed: CheckCircle,
  Failed: XCircle,
}

export function SessionList({ namespace, userPermission }: SessionListProps) {
  const [state, setState] = useState<SessionListState>({
    sessions: [],
    loading: true,
    error: null,
    hasMore: false,
  })
  const [actionLoading, setActionLoading] = useState<{ [key: string]: string }>({})
  const [filters, setFilters] = useState<FilterOptions>({
    phase: 'all',
    framework: 'all',
    searchTerm: '',
  })
  const [showFilters, setShowFilters] = useState(false)

  const fetchSessions = async (pageToken?: string, reset = false) => {
    try {
      if (reset) {
        setState(prev => ({ ...prev, loading: true, error: null }))
      }

      const token = localStorage.getItem('authToken')
      const queryParams = new URLSearchParams()

      if (pageToken) queryParams.set('pageToken', pageToken)
      if (filters.phase !== 'all') queryParams.set('phase', filters.phase)
      if (filters.framework !== 'all') queryParams.set('framework', filters.framework)
      if (filters.searchTerm) queryParams.set('search', filters.searchTerm)

      const response = await fetch(`/api/v1/namespaces/${namespace}/sessions?${queryParams}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      })

      if (!response.ok) {
        if (response.status === 403) {
          throw new Error(`Access denied to namespace: ${namespace}`)
        }
        throw new Error(`Failed to fetch sessions: ${response.statusText}`)
      }

      const data: SessionListResponse = await response.json()

      setState(prev => ({
        ...prev,
        sessions: reset ? data.sessions : [...prev.sessions, ...data.sessions],
        loading: false,
        error: null,
        hasMore: data.hasMore,
        nextPageToken: data.nextPageToken,
      }))
    } catch (err) {
      setState(prev => ({
        ...prev,
        loading: false,
        error: err instanceof Error ? err.message : 'Unknown error',
      }))
    }
  }

  useEffect(() => {
    fetchSessions(undefined, true)

    // Poll for updates every 15 seconds
    const interval = setInterval(() => fetchSessions(undefined, true), 15000)
    return () => clearInterval(interval)
  }, [namespace, filters])

  const handleRefresh = () => {
    fetchSessions(undefined, true)
  }

  const handleLoadMore = () => {
    if (state.nextPageToken && !state.loading) {
      fetchSessions(state.nextPageToken)
    }
  }

  const handleSessionAction = async (sessionName: string, action: 'stop' | 'delete') => {
    if (userPermission === 'viewer' && action === 'delete') {
      return // Viewers cannot delete sessions
    }

    try {
      setActionLoading(prev => ({ ...prev, [sessionName]: action }))

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

      // Refresh the session list
      fetchSessions(undefined, true)
    } catch (err) {
      console.error(`Failed to ${action} session:`, err)
      // TODO: Show toast notification
    } finally {
      setActionLoading(prev => {
        const newState = { ...prev }
        delete newState[sessionName]
        return newState
      })
    }
  }

  const filteredSessions = state.sessions.filter(session => {
    if (filters.phase !== 'all' && session.status?.phase !== filters.phase) {
      return false
    }
    if (filters.framework !== 'all' && session.spec.framework.type !== filters.framework) {
      return false
    }
    if (filters.searchTerm) {
      const searchLower = filters.searchTerm.toLowerCase()
      const sessionName = getSessionDisplayName(session).toLowerCase()
      const description = getSessionDescription(session).toLowerCase()
      if (!sessionName.includes(searchLower) && !description.includes(searchLower)) {
        return false
      }
    }
    return true
  })

  const getPhaseIcon = (phase: SessionPhase) => {
    const Icon = PHASE_ICONS[phase]
    return Icon ? <Icon className="h-4 w-4" /> : null
  }

  if (state.loading && state.sessions.length === 0) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-12">
          <div className="flex items-center space-x-2">
            <RefreshCw className="h-5 w-5 animate-spin" />
            <span>Loading sessions...</span>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (state.error) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-12">
          <div className="text-center">
            <AlertTriangle className="h-8 w-8 text-red-500 mx-auto mb-2" />
            <p className="text-red-600 mb-4">{state.error}</p>
            <Button onClick={handleRefresh} variant="outline" size="sm">
              <RefreshCw className="h-4 w-4 mr-2" />
              Retry
            </Button>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-4">
      {/* Header with actions */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">Sessions</h2>
          <p className="text-sm text-gray-500">
            {filteredSessions.length} session{filteredSessions.length !== 1 ? 's' : ''} in {namespace}
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <Button
            onClick={() => setShowFilters(!showFilters)}
            variant="outline"
            size="sm"
          >
            <Filter className="h-4 w-4 mr-2" />
            Filters
          </Button>
          <Button onClick={handleRefresh} variant="outline" size="sm" disabled={state.loading}>
            <RefreshCw className={`h-4 w-4 mr-2 ${state.loading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
        </div>
      </div>

      {/* Filters panel */}
      {showFilters && (
        <Card>
          <CardContent className="pt-6">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Search
                </label>
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-400" />
                  <input
                    type="text"
                    placeholder="Search sessions..."
                    value={filters.searchTerm}
                    onChange={(e) => setFilters(prev => ({ ...prev, searchTerm: e.target.value }))}
                    className="w-full pl-10 pr-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Status
                </label>
                <select
                  value={filters.phase}
                  onChange={(e) => setFilters(prev => ({ ...prev, phase: e.target.value as SessionPhase | 'all' }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="all">All Statuses</option>
                  <option value="Pending">Pending</option>
                  <option value="Running">Running</option>
                  <option value="Completed">Completed</option>
                  <option value="Failed">Failed</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Framework
                </label>
                <select
                  value={filters.framework}
                  onChange={(e) => setFilters(prev => ({ ...prev, framework: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="all">All Frameworks</option>
                  <option value="claude-code">Claude Code</option>
                  <option value="generic">Generic</option>
                </select>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Sessions table */}
      <Card>
        <CardContent className="p-0">
          {filteredSessions.length === 0 ? (
            <div className="text-center py-12">
              <div className="text-gray-500 mb-4">
                No sessions found{filters.searchTerm || filters.phase !== 'all' || filters.framework !== 'all' ? ' matching your filters' : ''}
              </div>
              {userPermission === 'editor' && (
                <Link href={`/namespace/${namespace}/new`}>
                  <Button>Create New Session</Button>
                </Link>
              )}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Session</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Framework</TableHead>
                  <TableHead>Duration</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="w-10"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredSessions.map((session) => (
                  <TableRow key={session.metadata.uid}>
                    <TableCell>
                      <div>
                        <Link
                          href={`/namespace/${namespace}/session/${session.metadata.name}`}
                          className="font-medium text-blue-600 hover:text-blue-800"
                        >
                          {getSessionDisplayName(session)}
                        </Link>
                        <div className="text-sm text-gray-500">
                          {getSessionDescription(session)}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant="secondary"
                        className={`${getPhaseColor(session.status?.phase || 'Pending')} flex items-center gap-1 w-fit`}
                      >
                        {getPhaseIcon(session.status?.phase || 'Pending')}
                        {session.status?.phase || 'Pending'}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center space-x-2">
                        <span className="text-sm">{session.spec.framework.type}</span>
                        <span className="text-xs text-gray-500">v{session.spec.framework.version}</span>
                      </div>
                    </TableCell>
                    <TableCell className="text-sm text-gray-500">
                      {formatDuration(
                        session.status?.startTime,
                        session.status?.completionTime
                      )}
                    </TableCell>
                    <TableCell className="text-sm text-gray-500">
                      {formatDistanceToNow(new Date(session.metadata.creationTimestamp), { addSuffix: true })}
                    </TableCell>
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem asChild>
                            <Link href={`/namespace/${namespace}/session/${session.metadata.name}`}>
                              <Eye className="h-4 w-4 mr-2" />
                              View Details
                            </Link>
                          </DropdownMenuItem>

                          {session.status?.phase === 'Running' && userPermission === 'editor' && (
                            <>
                              <DropdownMenuSeparator />
                              <DropdownMenuItem
                                onClick={() => handleSessionAction(session.metadata.name, 'stop')}
                                disabled={actionLoading[session.metadata.name] === 'stop'}
                                className="text-orange-600"
                              >
                                <Square className="h-4 w-4 mr-2" />
                                Stop Session
                              </DropdownMenuItem>
                            </>
                          )}

                          {userPermission === 'editor' && (
                            <>
                              <DropdownMenuSeparator />
                              <DropdownMenuItem
                                onClick={() => handleSessionAction(session.metadata.name, 'delete')}
                                disabled={actionLoading[session.metadata.name] === 'delete'}
                                className="text-red-600"
                              >
                                <Trash2 className="h-4 w-4 mr-2" />
                                Delete
                              </DropdownMenuItem>
                            </>
                          )}
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}

          {/* Load more button */}
          {state.hasMore && (
            <div className="flex justify-center py-4 border-t">
              <Button
                onClick={handleLoadMore}
                variant="outline"
                disabled={state.loading}
              >
                {state.loading ? (
                  <>
                    <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                    Loading...
                  </>
                ) : (
                  'Load More'
                )}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}