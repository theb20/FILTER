"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { adminAct, errorMessage } from "@/lib/admin";
import type { FormState } from "@/components/ui/client";

export async function setCustomerStatus(id: string, status: "active" | "suspended"): Promise<FormState> {
  const r = await adminAct("PATCH", `/admin/customers/${id}`, { body: { status } });
  if (!r.ok) return { error: errorMessage(r) };
  revalidatePath("/customers");
  redirect(`/customers/${id}?ok=${status === "suspended" ? "suspended" : "reactivated"}`);
}
