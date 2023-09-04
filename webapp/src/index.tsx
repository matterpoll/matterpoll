import MatterPollPlugin from '@/plugin';
import manifest from '@/manifest';

declare global {
    interface Window {
        registerPlugin(id: string, plugin: MatterPollPlugin): void
    }
}

window.registerPlugin(manifest.id, new MatterPollPlugin());
