import { useEffect } from "react";
import { useNavigate } from "react-router";
import type { Route } from "./+types/_index";

export default function Home() {
  const navigate = useNavigate();
  
  useEffect(() => {
    navigate("/train/453", { replace: true });
  }, [navigate]);

  // This component won't actually render due to the redirect
  return null;
}
