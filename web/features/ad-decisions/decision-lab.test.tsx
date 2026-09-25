import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createAdDecision, explainAdDecision } from "./actions";
import { DecisionLab } from "./decision-lab";

vi.mock("./actions", () => ({ createAdDecision: vi.fn(), explainAdDecision: vi.fn() }));

const placements = [{ code: "search_results", display_name: "Search results" }];
const inputID = "51000000-0000-4000-8000-000000000001";
const decision = {
  decision_id: "51000000-0000-4000-8000-000000000002",
  opportunity_id: inputID,
  placement: "search_results",
  country: "US",
  outcome: "FILL" as const,
  selection: { campaign_id: "51000000-0000-4000-8000-000000000003", campaign_version: 2 },
};

function completeForm(country = "us") {
  fireEvent.change(screen.getByLabelText("Opportunity ID"), { target: { value: inputID } });
  fireEvent.change(screen.getByLabelText("Country"), { target: { value: country } });
  fireEvent.click(screen.getByRole("button", { name: "Request decision" }));
}

describe("DecisionLab", () => {
  beforeEach(() => vi.clearAllMocks());

  it("validates the opportunity before submission", async () => {
    render(<DecisionLab placements={placements} />);
    fireEvent.click(screen.getByRole("button", { name: "Request decision" }));
    expect(await screen.findByText("Use a canonical lowercase UUID")).toBeInTheDocument();
    expect(createAdDecision).not.toHaveBeenCalled();
  });

  it("renders a fill and normalized values", async () => {
    vi.mocked(createAdDecision).mockResolvedValue({ ok: true, decision });
    render(<DecisionLab placements={placements} />);
    completeForm();
    expect(await screen.findByRole("heading", { name: "FILL" })).toBeInTheDocument();
    expect(screen.getByText(decision.selection.campaign_id)).toBeInTheDocument();
    expect(screen.getByText("US")).toBeInTheDocument();
  });

  it("shows an in-flight loading state", async () => {
    let finish: ((value: { ok: true; decision: typeof decision }) => void) | undefined;
    vi.mocked(createAdDecision).mockImplementation(() => new Promise((resolve) => { finish = resolve; }));
    render(<DecisionLab placements={placements} />);
    completeForm();
    expect(await screen.findByRole("button", { name: "Deciding…" })).toBeDisabled();
    finish?.({ ok: true, decision });
    expect(await screen.findByRole("heading", { name: "FILL" })).toBeInTheDocument();
  });

  it("renders a no-fill without inventing an error", async () => {
    vi.mocked(createAdDecision).mockResolvedValue({ ok: true, decision: { ...decision, outcome: "NO_FILL", selection: null } });
    render(<DecisionLab placements={placements} />);
    completeForm();
    expect(await screen.findByRole("heading", { name: "NO_FILL" })).toBeInTheDocument();
    expect(screen.getByText("No eligible campaign")).toBeInTheDocument();
  });

  it("shows backend unavailability as an explicit state", async () => {
    vi.mocked(createAdDecision).mockResolvedValue({ ok: false, message: "Database operation could not be completed", fields: {}, unavailable: true });
    render(<DecisionLab placements={placements} />);
    completeForm();
    expect(await screen.findByRole("alert")).toHaveTextContent("Database operation could not be completed");
  });

  it("shows disabled diagnostics and enabled explanations", async () => {
    vi.mocked(createAdDecision).mockResolvedValue({ ok: true, decision });
    vi.mocked(explainAdDecision).mockResolvedValueOnce({ ok: false, message: "Diagnostic explanations are disabled on the backend.", fields: {}, unavailable: false, diagnosticsDisabled: true });
    render(<DecisionLab placements={placements} />);
    completeForm();
    fireEvent.click(await screen.findByRole("button", { name: "Explain decision" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("disabled");

    vi.mocked(explainAdDecision).mockResolvedValue({ ok: true, diagnostic: { decision, truncated: false, explanations: [{ campaign_id: decision.selection.campaign_id, campaign_version: 2, eligible: true, reasons: [] }] } });
    fireEvent.click(screen.getByRole("button", { name: "Explain decision" }));
    await waitFor(() => expect(screen.getByRole("heading", { name: "Diagnostic explanation" })).toBeInTheDocument());
    expect(screen.getByText("eligible")).toBeInTheDocument();
  });
});
