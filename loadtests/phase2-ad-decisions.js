import http from "k6/http";
import { check } from "k6";
import { Rate } from "k6/metrics";

const rate = Number(__ENV.RATE || "10");
const warmupSeconds = Number(__ENV.WARMUP_SECONDS || "30");
const measuredSeconds = Number(__ENV.MEASURED_SECONDS || "120");
const cohort = __ENV.COHORT || "mixed";
const endpoint = `${__ENV.BASE_URL || "http://host.docker.internal:18080"}/v1/ad-decisions`;
const countries = { zero: "XA", one: "XB", ten: "XC", hundred: "XD", thousand: "XE" };
const cohortNames = Object.keys(countries);
const noFill = new Rate("decision_no_fill");

export const options = {
  discardResponseBodies: false,
	summaryTrendStats: ["min", "med", "p(95)", "p(99)", "max", "avg"],
	// These tautological expressions materialize measured-period submetrics in
	// the JSON summary; they are reporting aids, not performance thresholds.
	thresholds: {
		"http_req_duration{period:measured}": ["max>=0"],
		"http_req_failed{period:measured}": ["rate>=0"],
		"http_reqs{period:measured}": ["count>=0"],
		"decision_no_fill{period:measured}": ["rate>=0"],
	},
  scenarios: {
    warmup: {
      executor: "constant-arrival-rate", rate, timeUnit: "1s", duration: `${warmupSeconds}s`,
      preAllocatedVUs: Math.max(20, rate), maxVUs: Math.max(100, rate * 4),
      tags: { period: "warmup" },
    },
    measured: {
      executor: "constant-arrival-rate", rate, timeUnit: "1s", duration: `${measuredSeconds}s`, startTime: `${warmupSeconds}s`,
      preAllocatedVUs: Math.max(20, rate), maxVUs: Math.max(100, rate * 4),
      tags: { period: "measured" },
    },
  },
};

function uuidForIteration() {
  const value = (__VU * 100000000 + __ITER + 1).toString(16).padStart(12, "0").slice(-12);
  return `a0000000-0000-4000-8000-${value}`;
}

export default function () {
  const selectedCohort = cohort === "mixed" ? cohortNames[__ITER % cohortNames.length] : cohort;
  const country = countries[selectedCohort];
  if (!country) throw new Error(`unsupported COHORT ${selectedCohort}`);
  const response = http.post(endpoint, JSON.stringify({
    opportunity_id: uuidForIteration(), placement: "search_results", country,
  }), { headers: { "Content-Type": "application/json" }, tags: { cohort: selectedCohort } });
  const valid = check(response, { "decision status is 200": (item) => item.status === 200 });
  if (valid) {
    const result = response.json();
    noFill.add(result.outcome === "NO_FILL", { cohort: selectedCohort });
    check(result, {
      "decision shape is valid": (item) => item.decision_id && item.opportunity_id && (item.outcome === "FILL" || item.outcome === "NO_FILL"),
    });
  }
}

export function handleSummary(data) {
  const path = __ENV.SUMMARY_PATH || `loadtests/results/${cohort}-${rate}.json`;
  return { [path]: JSON.stringify(data, null, 2), stdout: "" };
}
