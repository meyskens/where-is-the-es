import {
  isRouteErrorResponse,
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
} from "react-router";

import type { Route } from "./+types/root";
import "./app.css";

export const links: Route.LinksFunction = () => [
  { rel: "preconnect", href: "https://fonts.googleapis.com" },
  {
    rel: "preconnect",
    href: "https://fonts.gstatic.com",
    crossOrigin: "anonymous",
  },
  {
    rel: "stylesheet",
    href: "https://fonts.googleapis.com/css2?family=Inter:ital,opsz,wght@0,14..32,100..900;1,14..32,100..900&display=swap",
  },
];

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <Meta />
        <Links />
      </head>
      <body className="min-h-screen bg-[#CDC4E1] flex flex-col">
        <header className="bg-[#40002C] text-white shadow-md">
          <div className="container mx-auto px-4 py-3">
            <h1 className="text-xl font-bold">Where is the European Sleeper?</h1>
          </div>
        </header>
        
        <main className="container mx-auto px-4 py-6 flex-grow">
          <div className="bg-white rounded-lg-lg p-6">
            {children}
          </div>
        </main>
        
        <footer className="text-center text-sm text-black p-4 bg-[#CDC4E1] mt-auto">
          <div className="container mx-auto px-4">
            <p>
              Made with ❤️ by in Belgium, the Netherlands, Germany and the Czech Republic.
            </p>
            <p className="mt-2">
              Open source under the AGPL license. <a href="https://github.com/meyskens/where-is-the-es">View on GitHub</a>.
            </p>
          </div>
        </footer>
        
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  );
}

export default function App() {
  return <Outlet />;
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  let message = "Oops!";
  let details = "An unexpected error occurred.";
  let stack: string | undefined;

  if (isRouteErrorResponse(error)) {
    message = error.status === 404 ? "404" : "Error";
    details =
      error.status === 404
        ? "The requested page could not be found."
        : error.statusText || details;
  } else if (import.meta.env.DEV && error && error instanceof Error) {
    details = error.message;
    stack = error.stack;
  }

  return (
    <main className="pt-16 p-4 container mx-auto">
      <h1>{message}</h1>
      <p>{details}</p>
      {stack && (
        <pre className="w-full p-4 overflow-x-auto">
          <code>{stack}</code>
        </pre>
      )}
    </main>
  );
}
