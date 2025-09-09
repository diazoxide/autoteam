export function a11yProps(index: number) {
  return {
    id: `worker-tab-${index}`,
    'aria-controls': `worker-tabpanel-${index}`,
  };
}