import fs from 'fs/promises';
import path from 'path';
import { StitchToolClient, Stitch } from '@google/stitch-sdk';

const accessToken = process.env.STITCH_ACCESS_TOKEN;
const cloudProjectId = process.env.GOOGLE_CLOUD_PROJECT || 'app-companion-430619';
if (!accessToken) {
  throw new Error('Missing STITCH_ACCESS_TOKEN');
}

const projectId = '7371947775132871122';
const screens = [
  { id: 'b47bc6d4b17d41f49040efc4c790edaa', title: 'my-attendance-month-view' },
  { id: '792b5386be2e418ebb11961eec12d883', title: 'my-attendance-week-view' },
  { id: 'd527aa6030db4eb0ab0440d8aec0d190', title: 'team-attendance-month-view' },
  { id: '67417f5e41e8449199c9a5fb5832241a', title: 'detailed-attendance-report-admin-view' },
  { id: '6c59bb4a7f3c4975a6f5b521acc0a63d', title: 'my-attendance-day-view' },
  { id: '8b75d11e0b354022b87f4ccdb48a726c', title: 'team-attendance-week-view' },
  { id: 'c009797430504244b0f984b4a52d5f7f', title: 'employee-profile-redesigned' },
  { id: 'dbfaa96e02a5447d80244f0e5a2b8976', title: 'team-attendance-day-view' },
  { id: 'c1f0c00e60d54bf2a0ccbdb326b790d2', title: 'admin-dashboard-light' },
  { id: '6bdc2e36f6f241928f856c46076ed762', title: 'employee-home-light' },
  { id: 'ca19c5c61e364811a3c6bda6ea2c3c03', title: 'login-light' },
];

const outDir = path.resolve('downloads');
await fs.mkdir(outDir, { recursive: true });

const client = new StitchToolClient({ accessToken, projectId: cloudProjectId });
const stitch = new Stitch(client);
const project = stitch.project(projectId);

const manifest = [];
for (const entry of screens) {
  const screen = await project.getScreen(entry.id);
  const imageUrl = await screen.getImage();
  const htmlUrl = await screen.getHtml();
  manifest.push({ ...entry, imageUrl, htmlUrl });
}

await fs.writeFile(path.join(outDir, 'manifest.json'), JSON.stringify(manifest, null, 2));
await client.close();
console.log('manifest-ready');
