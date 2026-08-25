import type {Reducer} from 'redux';

export type UniqueIdentifier = string;
export type ReactResolvable = React.ReactNode | React.ElementType;

export interface PluginRegistry {

    /**
     * Register a component to render a custom body for posts with a specific type.
     * Custom post types must be prefixed with 'custom_'.
     */
    registerPostTypeComponent(typeName: string, component: ReactResolvable): UniqueIdentifier;

    /**
     * Unregister a component that provided a custom body for posts with a specific type.
     */
    unregisterPostTypeComponent(componentId: UniqueIdentifier): void;

    /**
     * Register a reducer against the Redux store. It will be accessible in redux state
     * under "state['plugins-<yourpluginid>']".
     */
    registerReducer(reducer: Reducer): void;

    /**
     * Register a handler for WebSocket events. Plugin events have "custom_<pluginid>_" prepended.
     */
    registerWebSocketEventHandler<T = Record<string, unknown>>(event: string, handler: (msg: {data: T}) => void): void;

    /**
     * Register a custom React component to manage the plugin configuration for the given setting key.
     */
    registerAdminConsoleCustomSetting(key: string, component: ReactResolvable, options?: {showTitle?: boolean}): void;

    // Add more if needed from https://developers.mattermost.com/extend/plugins/webapp/reference
}

/**
 * Subset of the webapp's `formatText` options that this plugin passes through.
 * The webapp exposes these helpers on `window.PostUtils` for plugins to reuse.
 */
export type FormatTextOptions = {
    atMentions?: boolean;
    mentionHighlight?: boolean;
    markdown?: boolean;
    autoLinkedUrlSchemes?: string[];
    renderer?: unknown;
};

export type MessageHtmlToComponentOptions = {
    emoji?: boolean;
};

declare global {
    interface Window {
        PostUtils: {
            formatText: (text: string, options?: FormatTextOptions) => string;
            messageHtmlToComponent: (
                html: string,
                isRHS?: boolean,
                options?: MessageHtmlToComponentOptions,
            ) => React.ReactNode;
        };
    }
}
