import http from "k6/http";
import { check } from "k6";
import { Counter, Trend } from "k6/metrics";

const baseURL = __ENV.BASE_URL || "http://host.docker.internal:18080";
const mode = __ENV.MODE || "hot";
const vus = Number(__ENV.VUS || "4");
const iterations = Number(__ENV.ITERATIONS || "2000");
const duration = __ENV.DURATION || "15s";
const runID = __ENV.RUN_ID || "run";
const hotID = "91000000-0000-4000-8000-000000000001";
const replayID = "91000000-0000-4000-8000-000000000002";
const insufficientID = "91000000-0000-4000-8000-000000000003";

const consumptionDuration = new Trend("consumption_duration", true);
const decisionDuration = new Trend("decision_duration", true);
const approved = new Counter("consumption_approved");
const insufficient = new Counter("consumption_insufficient");
const replayed = new Counter("consumption_replayed");
const failures = new Counter("workload_failures");

const mixed = mode === "mixed_hot" || mode === "mixed_spread";
const decisionOnly = mode === "decision_only";
export const options = {
  discardResponseBodies: false,
  summaryTrendStats: ["min", "med", "p(95)", "p(99)", "max", "avg"],
  scenarios: mixed ? {
    accounting: { executor: "constant-vus", vus, duration, exec: "accountingWork" },
    decisions: { executor: "constant-vus", vus, duration, exec: "decisionWork" },
  } : decisionOnly ? {
    decisions: { executor: "constant-vus", vus, duration, exec: "decisionWork" },
  } : {
    accounting: { executor: "shared-iterations", vus, iterations, maxDuration: "2m", exec: "accountingWork" },
  },
};

function spreadCampaign() {
  const suffix = (256 + (__ITER % 100)).toString(16).padStart(12, "0");
  return `91000000-0000-4000-8000-${suffix}`;
}

function opportunityID() {
  const suffix = (__VU * 100000000 + __ITER + 1).toString(16).padStart(12, "0").slice(-12);
  return `b0000000-0000-4000-8000-${suffix}`;
}

export function accountingWork() {
  let campaignID = hotID;
  if (mode === "spread" || mode === "mixed_spread") campaignID = spreadCampaign();
  if (mode === "replay") campaignID = replayID;
  if (mode === "insufficient") campaignID = insufficientID;
  const key = mode === "replay" ? `phase3-replay-key-${runID}` : `${mode}-${runID}-${__VU}-${__ITER}`;
  const started = Date.now();
  const response = http.post(`${baseURL}/v1/campaigns/${campaignID}/budget-consumptions`, JSON.stringify({
    amount_minor: "1", currency: "EUR",
  }), { headers: { "Content-Type": "application/json", "Idempotency-Key": key }, tags: { operation: "consume", mode } });
  consumptionDuration.add(Date.now() - started, { mode });
  if (!check(response, { "consumption status is 200": (item) => item.status === 200 })) {
    failures.add(1, { operation: "consume", mode });
    return;
  }
  const result = response.json();
  if (result.outcome === "APPROVED") approved.add(1, { mode });
  if (result.outcome === "INSUFFICIENT_BUDGET") insufficient.add(1, { mode });
  if (response.headers["Idempotency-Replayed"] === "true") replayed.add(1, { mode });
}

export function decisionWork() {
  const started = Date.now();
  const response = http.post(`${baseURL}/v1/ad-decisions`, JSON.stringify({
    opportunity_id: opportunityID(), placement: "search_results", country: "XE",
  }), { headers: { "Content-Type": "application/json" }, tags: { operation: "decision", mode } });
  decisionDuration.add(Date.now() - started, { mode });
  if (!check(response, { "decision status is 200": (item) => item.status === 200 })) {
    failures.add(1, { operation: "decision", mode });
  }
}

export function handleSummary(data) {
  const path = __ENV.SUMMARY_PATH || `loadtests/results/phase3-${mode}-c${vus}.json`;
  return { [path]: JSON.stringify(data, null, 2), stdout: "" };
}
