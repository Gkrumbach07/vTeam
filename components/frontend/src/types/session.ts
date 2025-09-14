// Multi-tenant Session CRD types aligned with data-model.md

export type SessionPhase = "Pending" | "Running" | "Completed" | "Failed";

export type TriggerSource = "github" | "jira" | "slack" | "manual";

export type FrameworkType = "claude-code" | "generic";

export type ConditionType =
  | "PolicyValidated"
  | "WorkloadCreated"
  | "WorkloadRunning"
  | "ArtifactsStored";

export type ConditionStatus = "True" | "False" | "Unknown";

export interface Trigger {
  source: TriggerSource;
  event: string;
  payload?: Record<string, any>;
}

export interface Framework {
  type: FrameworkType;
  version: string;
  config?: Record<string, any>;
}

export interface ModelConstraints {
  allowed: string[];
  budget: string;
}

export interface ToolConstraints {
  allowed: string[];
}

export interface Policy {
  modelConstraints: ModelConstraints;
  toolConstraints: ToolConstraints;
  approvalRequired: boolean;
}

export interface ArtifactReference {
  name: string;
  type: string;
  location: string;
}

export interface ArtifactStorage {
  type: "s3" | "pvc" | "external";
  location: string;
}

export interface Artifacts {
  references: ArtifactReference[];
  storage: ArtifactStorage;
}

export interface SessionCondition {
  type: ConditionType;
  status: ConditionStatus;
  lastTransitionTime: string;
  reason?: string;
  message?: string;
}

export interface HistoryEvent {
  timestamp: string;
  event: string;
  data?: Record<string, any>;
}

export interface SessionSpec {
  trigger: Trigger;
  framework: Framework;
  policy: Policy;
  artifacts?: Artifacts;
}

export interface SessionStatus {
  phase: SessionPhase;
  conditions: SessionCondition[];
  history: HistoryEvent[];
  startTime?: string;
  completionTime?: string;
}

export interface SessionMetadata {
  name: string;
  namespace: string;
  creationTimestamp: string;
  uid: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

export interface Session {
  apiVersion: "ambient.ai/v1alpha1";
  kind: "Session";
  metadata: SessionMetadata;
  spec: SessionSpec;
  status?: SessionStatus;
}

// Request types for API calls
export interface CreateSessionRequest {
  trigger?: Partial<Trigger>;
  framework: Framework;
  policy?: Partial<Policy>;
  artifacts?: Partial<Artifacts>;
}

export interface UpdateSessionRequest {
  spec?: Partial<SessionSpec>;
}

// Response types for API calls
export interface SessionListResponse {
  sessions: Session[];
  totalCount: number;
  hasMore: boolean;
  nextPageToken?: string;
}

// Namespace and user types
export interface UserNamespace {
  namespace: string;
  permission: "viewer" | "editor";
  policy?: {
    budget?: string;
    sessionsActive?: number;
    sessionsMax?: number;
    retention?: string;
  };
}

export interface UserNamespacesResponse {
  namespaces: UserNamespace[];
}

// Framework configuration types
export interface FrameworkConfig {
  [key: string]: any;
  model?: string;
  temperature?: number;
  maxTokens?: number;
  timeout?: number;
}

// Legacy types for backward compatibility (to be phased out)
export type AgenticSessionPhase = SessionPhase;
export type AgenticSession = Session;
export type AgenticSessionSpec = SessionSpec;
export type AgenticSessionStatus = SessionStatus;

// Utility functions
export function getSessionDisplayName(session: Session): string {
  return session.metadata.annotations?.['ambient.ai/display-name'] ||
         session.metadata.name;
}

export function getSessionDescription(session: Session): string {
  const trigger = session.spec.trigger;
  const framework = session.spec.framework;

  if (trigger.source === 'manual') {
    return `Manual ${framework.type} session`;
  }

  return `${trigger.source} ${trigger.event} → ${framework.type}`;
}

export function formatDuration(startTime?: string, endTime?: string): string {
  if (!startTime) return 'Not started';

  const start = new Date(startTime);
  const end = endTime ? new Date(endTime) : new Date();
  const durationMs = end.getTime() - start.getTime();

  const seconds = Math.floor(durationMs / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);

  if (hours > 0) {
    return `${hours}h ${minutes % 60}m`;
  }
  if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`;
  }
  return `${seconds}s`;
}

export function getPhaseColor(phase: SessionPhase): string {
  switch (phase) {
    case 'Pending':
      return 'text-yellow-600 bg-yellow-50 border-yellow-200';
    case 'Running':
      return 'text-blue-600 bg-blue-50 border-blue-200';
    case 'Completed':
      return 'text-green-600 bg-green-50 border-green-200';
    case 'Failed':
      return 'text-red-600 bg-red-50 border-red-200';
    default:
      return 'text-gray-600 bg-gray-50 border-gray-200';
  }
}