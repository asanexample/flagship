import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Route, Routes } from "react-router-dom";
import { Layout } from "./components/Layout";
import { FlagsPage } from "./features/flags/FlagsPage";
import { FlagDetail } from "./features/flags/FlagDetail";
import type { ProductRef } from "./lib/types";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 5_000, refetchOnWindowFocus: false, retry: 1 },
  },
});

// v1 is scoped to a single Product context (the first consumer). A Product switcher is a follow-up.
const PRODUCT: ProductRef = { team: "alpha", product: "shop" };

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Layout product={PRODUCT}>
          <Routes>
            <Route path="/" element={<FlagsPage product={PRODUCT} />} />
            <Route path="/flags/:key" element={<FlagDetail product={PRODUCT} />} />
          </Routes>
        </Layout>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
