module Main exposing (..)

import Html exposing (..)
import Http
import WebSocket


--import Json.Decode exposing (..)
--import Json.Decode.Pipeline exposing (..)


type alias Model =
    { rows : List String
    }


init : ( Model, Cmd msg )
init =
    ( { rows = [ "initial" ] }, Cmd.none )


type Msg
    = NewMessageFromServer String


update : Msg -> Model -> ( Model, Cmd msg )
update msg model =
    case msg of
        NewMessageFromServer s ->
            ( { model | rows = model.rows ++ [ s ] }, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions model =
    WebSocket.listen "wss://localhost:8080/api/server" NewMessageFromServer


view : Model -> Html msg
view model =
    let
        rowsAsHtml : List (Html msg)
        rowsAsHtml =
            List.map (\s -> div [] [ text s ]) model.rows
    in
        div [] rowsAsHtml


main =
    Html.program
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        }
