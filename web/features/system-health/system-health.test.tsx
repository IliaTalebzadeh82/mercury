import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { SystemHealth, SystemHealthLoading } from "./system-health";

describe("SystemHealth", () => {
  it("renders healthy state", () => {
    render(<SystemHealth health={{ kind: "healthy", detail: "Backend and PostgreSQL are ready" }} />);
    expect(screen.getByText("Healthy")).toBeInTheDocument();
  });

  it("renders database readiness failure as degradation", () => {
    render(<SystemHealth health={{ kind: "degraded", detail: "PostgreSQL is unavailable" }} />);
    expect(screen.getByText("Degraded")).toBeInTheDocument();
    expect(screen.getByText("PostgreSQL is unavailable")).toBeInTheDocument();
  });

  it("renders unavailable state", () => {
    render(<SystemHealth health={{ kind: "unavailable", detail: "Backend could not be reached" }} />);
    expect(screen.getByText("Unavailable")).toBeInTheDocument();
  });

  it("renders malformed frontend configuration explicitly", () => {
    render(<SystemHealth health={{ kind: "misconfigured", detail: "MERCURY_API_URL is invalid" }} />);
    expect(screen.getByText("Misconfigured")).toBeInTheDocument();
    expect(screen.getByText("MERCURY_API_URL is invalid")).toBeInTheDocument();
  });

  it("renders an accessible loading state", () => {
    render(<SystemHealthLoading />);
    expect(screen.getByLabelText("Loading system health")).toBeInTheDocument();
  });
});
