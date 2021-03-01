import PropTypes from 'prop-types';
import React from 'react';

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
    }

    getStatusColors = (theme) => {
        return {
            good: '#00c100',
            warning: '#dede01',
            danger: theme.errorTextColor,
            default: theme.centerChannelColor,
            primary: theme.buttonBg,
            success: theme.onlineIndicator,
        };
    }

    invertColor = (color) => {
        return color.match('^#(?:[0-9a-fA-F]{3}){1,2}$') ? invert(color) : color;
    }

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

        let customButtonStyle;
        if (action.style) {
            const STATUS_COLORS = this.getStatusColors(theme);
            const hexColor =
                STATUS_COLORS[action.style] ||
                theme[action.style] ||
                (action.style.match('^#(?:[0-9a-fA-F]{3}){1,2}$') && action.style);
            if (hexColor) {
                if (this.props.hasVoted) {
                    customButtonStyle = {
                        borderColor: changeOpacity(this.invertColor(hexColor), 0.25),
                        backgroundColor: changeOpacity(this.invertColor(theme.centerChannelBg), 0.75),
                        color: this.invertColor(hexColor),
                        borderWidth: 2,
                    };
                } else {
                    customButtonStyle = {
                        borderColor: changeOpacity(hexColor, 0.25),
                        backgroundColor: theme.centerChannelBg,
                        color: hexColor,
                        borderWidth: 2,
                    };
                }
            }
        }

        return (
            <button
                data-action-id={action.id}
                data-action-cookie={action.cookie}
                key={action.id}
                onClick={this.handleAction}
                style={customButtonStyle}
            >
                {message}
            </button>
        );
    }
}
