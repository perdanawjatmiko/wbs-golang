import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/lib/utils"
import { Slot } from "@radix-ui/react-slot"

const buttonVariants = cva(
  "inline-flex min-h-[42px] shrink-0 items-center justify-center gap-2 rounded-[8px] border px-[15px] py-[9px] text-sm font-semibold leading-[1.2] whitespace-nowrap transition-[background-color,border-color,box-shadow] outline-none focus-visible:outline-[3px] focus-visible:outline-offset-[3px] focus-visible:outline-ring disabled:cursor-not-allowed disabled:opacity-[.58] aria-invalid:border-destructive [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default: "border-transparent bg-primary text-white shadow-sm hover:bg-primary-hover",
        destructive:
          "border-transparent bg-danger text-white hover:bg-red-800 focus-visible:ring-danger/30",
        outline:
          "border-border bg-white text-slate-700 hover:border-slate-300 hover:bg-slate-50",
        secondary:
          "border-border bg-white text-slate-700 hover:border-slate-300 hover:bg-slate-50",
        ghost:
          "border-transparent bg-transparent px-2 hover:bg-slate-100 hover:text-slate-900",
        link: "min-h-0 border-transparent bg-transparent px-0 py-0 text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-[42px]",
        xs: "h-7 min-h-7 gap-1 rounded-[8px] px-2 text-xs [&_svg:not([class*='size-'])]:size-3",
        sm: "h-9 min-h-9 gap-1.5 rounded-[8px] px-3 text-sm",
        lg: "h-12 min-h-12 rounded-[8px] px-6",
        icon: "size-10 min-h-10 p-0",
        "icon-xs": "size-7 min-h-7 rounded-[8px] p-0 [&_svg:not([class*='size-'])]:size-3",
        "icon-sm": "size-9 min-h-9 p-0",
        "icon-lg": "size-12 min-h-12 p-0",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

function Button({
  className,
  variant = "default",
  size = "default",
  asChild = false,
  ...props
}: React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean
  }) {
  const Comp = asChild ? Slot : "button"

  return (
    <Comp
      data-slot="button"
      data-variant={variant}
      data-size={size}
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export { Button, buttonVariants }
