export type ActionResult = {
  ok: boolean;
  message?: string;
  fields?: Record<string, string>;
  stale?: boolean;
  resourceId?: string;
  resourceVersion?: number;
};
