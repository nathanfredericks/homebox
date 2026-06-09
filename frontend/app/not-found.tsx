import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import Link from "next/link";

export default function NotFound() {
  return (
    <Box sx={{ display: "flex", minHeight: "100vh", alignItems: "center", justifyContent: "center", p: 3 }}>
      <Stack spacing={2} alignItems="center" textAlign="center">
        <Typography variant="h2" component="h1">
          404
        </Typography>
        <Typography color="text.secondary">This page could not be found.</Typography>
        <Button variant="contained" component={Link} href="/home">
          Go home
        </Button>
      </Stack>
    </Box>
  );
}
