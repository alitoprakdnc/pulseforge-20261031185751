import React from 'react';
import { render, screen, cleanup } from '@testing-library/react';
import Dashboard from '../../../components/Dashboard';
import type { Pipeline } from '../../../types';

// Mock the SWR hook that Dashboard uses to fetch pipelines
jest.mock('../../hooks/usePipelines', () => ({
  __esModule: true,
  default: jest.fn(),
}));

// Import the mocked hook after jest.mock so TypeScript knows the shape
import usePipelines from '../../hooks/usePipelines';

describe('Dashboard component', () => {
  const mockPipelines: Pipeline[] = [
    {
      id: 'pipeline-1',
      name: 'Pipeline One',
      provider: 'github',
      repo: 'org/repo-one',
      // additional fields that the component might read
      createdAt: '2024-01-01T00:00:00Z',
      updatedAt: '2024-01-02T00:00:00Z',
    },
    {
      id: 'pipeline-2',
      name: 'Pipeline Two',
      provider: 'gitlab',
      repo: 'org/repo-two',
      createdAt: '2024-01-03T00:00:00Z',
      updatedAt: '2024-01-04T00:00:00Z',
    },
    {
      id: 'pipeline-3',
      name: 'Pipeline Three',
      provider: 'circleci',
      repo: 'org/repo-three',
      createdAt: '2024-01-05T00:00:00Z',
      updatedAt: '2024-01-06T00:00:00Z',
    },
  ];

  beforeEach(() => {
    // Reset mock implementation before each test
    (usePipelines as jest.Mock).mockReturnValue({
      pipelines: mockPipelines,
      isLoading: false,
      isError: false,
    });
  });

  afterEach(() => {
    cleanup();
    jest.clearAllMocks();
  });

  it('renders a PipelineCard for each pipeline returned by the hook', () => {
    render(<Dashboard />);

    // Verify that each pipeline name appears in the document
    mockPipelines.forEach((pipeline) => {
      expect(screen.getByText(pipeline.name)).toBeInTheDocument();
    });

    // Verify that the number of rendered cards matches the mock data length.
    // The PipelineCard component is expected to expose a data-testid="pipeline-card"
    const renderedCards = screen.getAllByTestId('pipeline-card');
    expect(renderedCards).toHaveLength(mockPipelines.length);
  });
});