import PropTypes from 'prop-types';
import React from 'react';
import styled, {css} from 'styled-components';

import {changeOpacity} from 'mattermost-redux/utils/theme_utils';
import invert from 'invert-color';

const PostUtils = window.PostUtils;

export default class ActionButton extends React.PureComponent {
    static propTypes = {
        action: PropTypes.object.isRequired,
        postId: PropTypes.string.isRequired,
        theme: PropTypes.object.isRequired,
        hasVoted: PropTypes.bool,

        actions: PropTypes.shape({
            voteAnswer: PropTypes.func.isRequired,
        }).isRequired,
    };

    getStatusColors = (theme) => {
        return {
            good: '#339970',
            warning: '#CC8F00',
            danger: theme.errorTextColor,
            default: theme.centerChannelColor,
            primary: theme.buttonBg,
            success: '#339970',
        };
    };

    handleAction = (e) => {
        e.preventDefault();
        const actionId = e.currentTarget.getAttribute('data-action-id');

        this.props.actions.voteAnswer(
            this.props.postId,
            actionId,
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
        let hexColor;
        if (action.style) {
            const STATUS_COLORS = this.getStatusColors(theme);
            hexColor =
                STATUS_COLORS[action.style] ||
                theme[action.style] ||
                (action.style.match('^#(?:[0-9a-fA-F]{3}){1,2}$') && action.style);
        }

        return (
            <ActionBtn
                data-action-id={action.id}
                data-action-cookie={action.cookie}
                key={action.id}
                onClick={this.handleAction}
                className='btn btn-sm'
                hexColor={hexColor}
                isVoted={this.props.hasVoted}
            >
                {message}
            </ActionBtn>
        );
    }
}

const ActionBtn = styled.button`
    ${({hexColor, isVoted}) => hexColor && css`
        background-color: ${changeOpacity(hexColor, isVoted ? 0.92 : 0.08)} !important;
        color: ${isVoted ? invert(hexColor) : hexColor} !important;
        &:hover {
            background-color: ${changeOpacity(hexColor, isVoted ? 0.88 : 0.12)} !important;
        }
        &:active {
            background-color: ${changeOpacity(hexColor, isVoted ? 0.84 : 0.16)} !important;
        }
    `}
`;