export function FeatureUnavailable({ message }: { message: string }) {
  return (
    <section role="alert" className="rounded-lg border border-amber-200 bg-amber-50 p-6">
      <h1 className="text-xl font-semibold text-amber-950">Campaign control plane unavailable</h1>
      <p className="mt-2 text-sm text-amber-900">{message}</p>
    </section>
  );
}
