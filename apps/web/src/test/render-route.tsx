import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
	createMemoryHistory,
	createRouter,
	RouterProvider,
} from "@tanstack/react-router";
import { render } from "@testing-library/react";
import { routeTree } from "~/routeTree.gen";

export function renderRoute(path: string) {
	const queryClient = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	const router = createRouter({
		routeTree,
		context: { queryClient },
		history: createMemoryHistory({ initialEntries: [path] }),
	});
	return render(
		<QueryClientProvider client={queryClient}>
			<RouterProvider router={router} />
		</QueryClientProvider>,
	);
}
