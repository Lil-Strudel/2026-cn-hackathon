import type { QueryClient } from "@tanstack/react-query";
import { createRootRouteWithContext, Outlet } from "@tanstack/react-router";
import { ThemeToggle } from "~/components/theme-toggle";

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()(
	{
		component: () => (
			<div className="min-h-dvh bg-background text-foreground">
				<header className="flex justify-end px-4 pt-4 sm:px-8">
					<ThemeToggle />
				</header>
				<Outlet />
			</div>
		),
	},
);
