import { Skeleton } from "@/components/ui/ui";

export default function Loading() {
  return (
    <div role="status" aria-live="polite" aria-label="Chargement">
      <Skeleton className="h-8 w-56" />
      <Skeleton className="mt-3 h-4 w-96 max-w-full" />
      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-28" />)}
      </div>
      <Skeleton className="mt-6 h-72" />
      <span className="sr-only">Chargement en cours…</span>
    </div>
  );
}
