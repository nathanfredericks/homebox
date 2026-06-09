"use client";

import { useTranslation } from "react-i18next";
import Box from "@mui/material/Box";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { WorkerBundleSpike } from "~~/lib/spikes/WorkerBundleSpike";

/**
 * Temporary placeholder. The auth-shell agent replaces this with the
 * login/register entry page. It exists so the Next.js scaffold renders and so
 * the foundation exit criteria (theme + an i18next-translated string) can be
 * verified.
 */
export default function HomePage() {
  const { t } = useTranslation();

  return (
    <Box sx={{ display: "flex", minHeight: "100vh", alignItems: "center", justifyContent: "center", p: 3 }}>
      <Stack spacing={1} alignItems="center" textAlign="center">
        <Typography variant="h3" component="h1" color="primary">
          Homebox
        </Typography>
        <Typography color="text.secondary">{t("home.quick_statistics")}</Typography>
      </Stack>
      <WorkerBundleSpike />
    </Box>
  );
}
