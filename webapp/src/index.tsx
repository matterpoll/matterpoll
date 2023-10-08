import MatterPollPlugin from '@/plugin';
import manifest from '@/manifest';

declare global {
    interface Window {
        registerPlugin(id: string, plugin: MatterPollPlugin): void
    }

    // fix for a type problem in webapp as of 6dcac2
    type DeepPartial<T> = {
        [P in keyof T]?: DeepPartial<T[P]>;
    }
}

window.registerPlugin(manifest.id, new MatterPollPlugin());
