export interface LogEvent {
  instanceId: string;
  level: string; // 'INFO' | 'WARN' | 'ERROR' | 'CHAT'
  line: string;
  timestamp: string;
}

export interface ProgressEvent {
  instanceId: string;
  stage: string;
  message: string;
  percentage: number;
  currentItem: string;
  completedItems: number;
  totalItems: number;
  speedKbps: number;
}
