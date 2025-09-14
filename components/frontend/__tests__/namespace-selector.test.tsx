import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { NamespaceSelector } from '@/components/ui/namespace-selector'

// Mock fetch for API calls
global.fetch = jest.fn()

describe('NamespaceSelector', () => {
  beforeEach(() => {
    ;(fetch as jest.Mock).mockClear()
  })

  test('displays available namespaces for user', async () => {
    // Mock API response for user namespaces
    ;(fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        namespaces: [
          { namespace: 'team-alpha', permission: 'editor' },
          { namespace: 'team-beta', permission: 'viewer' },
        ],
      }),
    })

    const mockOnSelect = jest.fn()

    render(<NamespaceSelector onSelect={mockOnSelect} />)

    // Wait for namespaces to load
    await waitFor(() => {
      expect(screen.getByText('team-alpha')).toBeInTheDocument()
      expect(screen.getByText('team-beta')).toBeInTheDocument()
    })

    // Verify editor badge is shown
    expect(screen.getByText('Editor')).toBeInTheDocument()
    expect(screen.getByText('Viewer')).toBeInTheDocument()
  })

  test('calls onSelect when namespace is chosen', async () => {
    ;(fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        namespaces: [{ namespace: 'team-alpha', permission: 'editor' }],
      }),
    })

    const mockOnSelect = jest.fn()

    render(<NamespaceSelector onSelect={mockOnSelect} />)

    await waitFor(() => {
      expect(screen.getByText('team-alpha')).toBeInTheDocument()
    })

    // Click on namespace
    fireEvent.click(screen.getByText('team-alpha'))

    expect(mockOnSelect).toHaveBeenCalledWith('team-alpha')
  })

  test('shows loading state while fetching namespaces', () => {
    ;(fetch as jest.Mock).mockImplementation(
      () => new Promise(resolve => setTimeout(resolve, 100))
    )

    render(<NamespaceSelector onSelect={jest.fn()} />)

    expect(screen.getByText('Loading namespaces...')).toBeInTheDocument()
  })

  test('shows error when namespace fetch fails', async () => {
    ;(fetch as jest.Mock).mockRejectedValueOnce(new Error('API Error'))

    render(<NamespaceSelector onSelect={jest.fn()} />)

    await waitFor(() => {
      expect(screen.getByText('Error loading namespaces')).toBeInTheDocument()
    })
  })

  test('filters namespaces based on search input', async () => {
    ;(fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        namespaces: [
          { namespace: 'team-alpha', permission: 'editor' },
          { namespace: 'team-beta', permission: 'viewer' },
          { namespace: 'project-gamma', permission: 'editor' },
        ],
      }),
    })

    render(<NamespaceSelector onSelect={jest.fn()} />)

    await waitFor(() => {
      expect(screen.getByText('team-alpha')).toBeInTheDocument()
    })

    // Search for "team"
    const searchInput = screen.getByPlaceholderText('Search namespaces...')
    fireEvent.change(searchInput, { target: { value: 'team' } })

    // Should show only team namespaces
    expect(screen.getByText('team-alpha')).toBeInTheDocument()
    expect(screen.getByText('team-beta')).toBeInTheDocument()
    expect(screen.queryByText('project-gamma')).not.toBeInTheDocument()
  })
})

// This component doesn't exist yet - test will fail
// Implementation will be added in T042