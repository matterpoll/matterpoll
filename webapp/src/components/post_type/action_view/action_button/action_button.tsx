import invert from 'invert-color';
import React from 'react';
import styled, {css} from 'styled-components';

import type {Theme} from 'mattermost-redux/selectors/entities/preferences';
import {changeOpacity} from 'mattermost-redux/utils/theme_utils';

import type {AttachmentAction} from '@/types/poll';

const PostUtils = window.PostUtils;

type Props = {
    action: AttachmentAction;
    postId: string;

    // Partial because the component tolerates a theme that is missing the colour a style names.
    theme: Partial<Theme>;
    hasVoted?: boolean;

    actions: {
        voteAnswer: (postId: string, actionId: string) => void;
    };
};

export default class ActionButton extends React.PureComponent<Props> {
    getStatusColors = (theme: Partial<Theme>): Record<string, string | undefined> => {
        return {
            good: '#339970',
            warning: '#CC8F00',
            danger: theme.errorTextColor,
            default: theme.centerChannelColor,
            primary: theme.buttonBg,
            success: '#339970',
        };
    };

    handleAction = (e: React.MouseEvent<HTMLButtonElement>) => {
        e.preventDefault();
        const actionId = e.currentTarget.getAttribute('data-action-id');

        this.props.actions.voteAnswer(
            this.props.postId,
            actionId as string,
        );
    };

    render() {
        const {action, theme} = this.props;

        const htmlFormattedText = PostUtils.formatText(action.name, {
            mentionHighlight: false,
            markdown: false,
            autoLinkedUrlSchemes: [],
        });
        const message = PostUtils.messageHtmlToComponent(htmlFormattedText, false, {emoji: true});
        let hexColor = '';
        if (action.style) {
            const STATUS_COLORS = this.getStatusColors(theme);
            hexColor =
                STATUS_COLORS[action.style] ||
                theme[action.style] ||
                (action.style.match('^#(?:[0-9a-fA-F]{3}){1,2}$') ? action.style : '');
        }

        return (
            <ActionBtn
                data-action-id={action.id}
                data-action-cookie={action.cookie}
                key={action.id}
                onClick={this.handleAction}
                className='btn btn-sm'
                $hexColor={hexColor}
                $isVoted={this.props.hasVoted}
            >
                {message}
            </ActionBtn>
        );
    }
}

const ActionBtn = styled.button<{$hexColor?: string; $isVoted?: boolean}>`
    ${({$hexColor, $isVoted}) => $hexColor && css`
        background-color: ${changeOpacity($hexColor, $isVoted ? 0.92 : 0.08)} !important;
        color: ${$isVoted ? invert($hexColor) : $hexColor} !important;
        &:hover {
            background-color: ${changeOpacity($hexColor, $isVoted ? 0.88 : 0.12)} !important;
        }
        &:active {
            background-color: ${changeOpacity($hexColor, $isVoted ? 0.84 : 0.16)} !important;
        }
    `}
`;
