import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import ErrorPage from "./error";

describe("ErrorPage", () => {
  afterEach(() => vi.restoreAllMocks());

  it("distinguishes a frontend rendering error and offers retry", () => {
    const logError = vi.spyOn(console, "error").mockImplementation(() => undefined);
    const reset = vi.fn();
    const error = Object.assign(new Error("sensitive render detail"), { digest: "stable-digest" });

    render(<ErrorPage error={error} reset={reset} />);

    expect(screen.getByRole("alert")).toHaveTextContent("Interface error");
    expect(logError).toHaveBeenCalledWith("Mercury frontend render failed", "stable-digest");
    expect(JSON.stringify(logError.mock.calls)).not.toContain("sensitive render detail");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(reset).toHaveBeenCalledOnce();
  });
});
