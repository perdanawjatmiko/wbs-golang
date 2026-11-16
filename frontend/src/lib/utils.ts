import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

/** Class composition helper used by generated and WBS-wrapped primitives. */
export function cn(...values: ClassValue[]): string {
  return twMerge(clsx(values));
}
