import "server-only";

import type { ActionResult } from "@/lib/action-result";
import { MercuryAPIError } from "./mercury";

export function actionError(error: unknown): ActionResult {
  if (error instanceof MercuryAPIError) {
    return { ok: false, message: error.message, fields: error.fields, stale: error.status === 412 };
  }
  return { ok: false, message: "The operation could not be completed" };
}
